// Package slip39 implements SLIP-0039 (Shamir's Secret Sharing for Mnemonic
// Codes), enough to split and recover an arbitrary master secret as
// human-friendly word shares.
//
// IMPORTANT for this project: the SLIP-39 *passphrase* used here is NOT the
// Trezor wallet passphrase. We use an empty SLIP-39 passphrase; the master
// secret we split is a random key-file (not a wallet seed). The Trezor
// passphrase is a separate secret stored only in the posthumous envelope.
//
// The implementation is spec-compliant so the produced shares are readable by
// any other SLIP-39 tool (Trezor, the reference library, ...). Correctness is
// verified against the official SLIP-39 test vectors (see slip39_test.go).
package slip39

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"strings"
)

// Spec constants.
const (
	radixBits          = 10
	idLengthBits       = 15
	iterationExpBits   = 4
	idExpLengthWords   = (idLengthBits + 1 + iterationExpBits) / radixBits // = 2
	checksumWords      = 3
	metadataWords      = idExpLengthWords + 2 + checksumWords // = 7
	minMnemonicWords   = 20                                   // metadata + ceil(128/10)
	digestLengthBytes  = 4
	baseIterationCount = 10000
	roundCount         = 4
	secretIndex        = 255
	digestIndex        = 254
	maxShareCount      = 16
)

// Errors.
var (
	ErrEmpty           = errors.New("slip39: no mnemonics provided")
	ErrInvalidWord     = errors.New("slip39: mnemonic contains a word not in the wordlist")
	ErrLength          = errors.New("slip39: invalid mnemonic length")
	ErrChecksum        = errors.New("slip39: invalid checksum")
	ErrPadding         = errors.New("slip39: invalid padding")
	ErrMismatch        = errors.New("slip39: mnemonics belong to different shares (mismatching id/exp/flags)")
	ErrGroupThreshold  = errors.New("slip39: mismatching group thresholds")
	ErrGroupCount      = errors.New("slip39: mismatching group counts")
	ErrThreshold       = errors.New("slip39: group threshold exceeds group count")
	ErrMemberThreshold = errors.New("slip39: mismatching member thresholds within a group")
	ErrDuplicate       = errors.New("slip39: duplicate member index within a group")
	ErrInsuffGroups    = errors.New("slip39: insufficient number of groups")
	ErrWrongGroups     = errors.New("slip39: wrong number of groups for the threshold")
	ErrWrongMembers    = errors.New("slip39: wrong number of member mnemonics for the threshold")
	ErrDigest          = errors.New("slip39: share digest mismatch (corrupted or wrong shares)")
	ErrSecretLength    = errors.New("slip39: master secret must be even length and at least 16 bytes")
	ErrParams          = errors.New("slip39: invalid threshold/count parameters")
)

// ---------------------------------------------------------------------------
// GF(256) arithmetic tables (reducing polynomial 0x11b, generator 3).
// ---------------------------------------------------------------------------

var gfExp [255]byte
var gfLog [256]int

func init() {
	poly := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(poly)
		gfLog[poly] = i
		poly = (poly << 1) ^ poly // multiply by 3 in GF(2)[x]
		if poly&0x100 != 0 {
			poly ^= 0x11b
		}
	}
	// gfLog[0] stays 0 (unused; the interpolation self-term contributes 0).
}

type share struct {
	index byte
	value []byte
}

// interpolate evaluates the Lagrange interpolation polynomial defined by the
// given (index, value) shares at point x, byte-wise over GF(256).
func interpolate(shares []share, x byte) []byte {
	n := len(shares[0].value)
	for _, s := range shares {
		if s.index == x {
			out := make([]byte, n)
			copy(out, s.value)
			return out
		}
	}
	logProd := 0
	for _, s := range shares {
		logProd += gfLog[int(s.index^x)]
	}
	result := make([]byte, n)
	for _, s := range shares {
		sum := 0
		for _, o := range shares {
			sum += gfLog[int(s.index^o.index)]
		}
		logBasis := mod255(logProd - gfLog[int(s.index^x)] - sum)
		for j := 0; j < n; j++ {
			v := s.value[j]
			if v != 0 {
				result[j] ^= gfExp[mod255(gfLog[int(v)]+logBasis)]
			}
		}
	}
	return result
}

func mod255(x int) int {
	x %= 255
	if x < 0 {
		x += 255
	}
	return x
}

// ---------------------------------------------------------------------------
// Digest + secret splitting / recovery.
// ---------------------------------------------------------------------------

func createDigest(randomData, secret []byte) []byte {
	mac := hmac.New(sha256.New, randomData)
	mac.Write(secret)
	return mac.Sum(nil)[:digestLengthBytes]
}

func splitSecret(threshold, count int, secret []byte) ([]share, error) {
	if threshold < 1 || count < 1 || threshold > count || count > maxShareCount {
		return nil, ErrParams
	}
	if threshold == 1 {
		out := make([]share, count)
		for i := 0; i < count; i++ {
			out[i] = share{byte(i), clone(secret)}
		}
		return out, nil
	}
	randomCount := threshold - 2
	out := make([]share, 0, count)
	base := make([]share, 0, threshold)
	for i := 0; i < randomCount; i++ {
		s := share{byte(i), randomBytes(len(secret))}
		out = append(out, s)
		base = append(base, s)
	}
	randomPart := randomBytes(len(secret) - digestLengthBytes)
	digest := createDigest(randomPart, secret)
	base = append(base, share{digestIndex, append(clone(digest), randomPart...)})
	base = append(base, share{secretIndex, clone(secret)})
	for i := randomCount; i < count; i++ {
		out = append(out, share{byte(i), interpolate(base, byte(i))})
	}
	return out, nil
}

func recoverSecret(threshold int, shares []share) ([]byte, error) {
	if threshold == 1 {
		return clone(shares[0].value), nil
	}
	secret := interpolate(shares, secretIndex)
	digestShare := interpolate(shares, digestIndex)
	digest := digestShare[:digestLengthBytes]
	randomPart := digestShare[digestLengthBytes:]
	if !hmac.Equal(digest, createDigest(randomPart, secret)) {
		return nil, ErrDigest
	}
	return secret, nil
}

// ---------------------------------------------------------------------------
// Master-secret encryption (4-round Feistel keyed by PBKDF2-HMAC-SHA256).
// ---------------------------------------------------------------------------

func roundFunction(i byte, passphrase []byte, e int, salt, r []byte) []byte {
	pw := append([]byte{i}, passphrase...)
	s := append(clone(salt), r...)
	iters := (baseIterationCount << e) / roundCount
	dk, err := pbkdf2.Key(sha256.New, string(pw), s, iters, len(r))
	if err != nil {
		panic("slip39: pbkdf2 failed: " + err.Error())
	}
	return dk
}

func getSalt(id int, extendable bool) []byte {
	if extendable {
		return []byte{}
	}
	return append([]byte("shamir"), byte(id>>8), byte(id&0xff))
}

func encrypt(masterSecret, passphrase []byte, e, id int, extendable bool) []byte {
	half := len(masterSecret) / 2
	l := clone(masterSecret[:half])
	r := clone(masterSecret[half:])
	salt := getSalt(id, extendable)
	for i := 0; i < roundCount; i++ {
		nr := xor(l, roundFunction(byte(i), passphrase, e, salt, r))
		l, r = r, nr
	}
	return append(clone(r), l...)
}

func decrypt(ems, passphrase []byte, e, id int, extendable bool) []byte {
	half := len(ems) / 2
	l := clone(ems[:half])
	r := clone(ems[half:])
	salt := getSalt(id, extendable)
	for i := roundCount - 1; i >= 0; i-- {
		nr := xor(l, roundFunction(byte(i), passphrase, e, salt, r))
		l, r = r, nr
	}
	return append(clone(r), l...)
}

// ---------------------------------------------------------------------------
// RS1024 checksum.
// ---------------------------------------------------------------------------

// rs1024Gen are the RS1024 generator coefficients. NOTE: the last value is
// 0x3f3f120 (NOT 0x3f3f4120, which some secondary sources list incorrectly).
// The official trezor/python-shamir-mnemonic reference and the test vectors
// require 0x3f3f120 — do not "fix" this back.
var rs1024Gen = []int{
	0xe0e040, 0x1c1c080, 0x3838100, 0x7070200, 0xe0e0009,
	0x1c0c2412, 0x38086c24, 0x3090fc48, 0x21b1f890, 0x3f3f120,
}

func rs1024Polymod(values []int) int {
	chk := 1
	for _, v := range values {
		b := chk >> 20
		chk = ((chk & 0xfffff) << 10) ^ v
		for i := 0; i < 10; i++ {
			if (b>>i)&1 == 1 {
				chk ^= rs1024Gen[i]
			}
		}
	}
	return chk
}

func customizationValues(extendable bool) []int {
	s := "shamir"
	if extendable {
		s = "shamir_extendable"
	}
	out := make([]int, len(s))
	for i := 0; i < len(s); i++ {
		out[i] = int(s[i])
	}
	return out
}

func rs1024Checksum(extendable bool, data []int) []int {
	values := append(append(customizationValues(extendable), data...), 0, 0, 0)
	polymod := rs1024Polymod(values) ^ 1
	return []int{(polymod >> 20) & 1023, (polymod >> 10) & 1023, polymod & 1023}
}

func rs1024Verify(extendable bool, words []int) bool {
	return rs1024Polymod(append(customizationValues(extendable), words...)) == 1
}

// ---------------------------------------------------------------------------
// Mnemonic encoding / decoding.
// ---------------------------------------------------------------------------

type rawShare struct {
	id              int
	extendable      bool
	iterationExp    int
	groupIndex      int
	groupThreshold  int // actual value (>=1)
	groupCount      int // actual value (>=1)
	memberIndex     int
	memberThreshold int // actual value (>=1)
	value           []byte
}

func encodeShare(s rawShare) (string, error) {
	bits := make([]int, 0, 200)
	bits = appendBits(bits, s.id, 15)
	bits = append(bits, boolBit(s.extendable))
	bits = appendBits(bits, s.iterationExp, 4)
	bits = appendBits(bits, s.groupIndex, 4)
	bits = appendBits(bits, s.groupThreshold-1, 4)
	bits = appendBits(bits, s.groupCount-1, 4)
	bits = appendBits(bits, s.memberIndex, 4)
	bits = appendBits(bits, s.memberThreshold-1, 4)

	valBits := bytesToBits(s.value)
	pad := (radixBits - (len(valBits) % radixBits)) % radixBits
	bits = append(bits, make([]int, pad)...) // left zero padding
	bits = append(bits, valBits...)

	dataWords := bitsToWords(bits)
	checksum := rs1024Checksum(s.extendable, dataWords)
	all := append(dataWords, checksum...)

	words := make([]string, len(all))
	for i, w := range all {
		words[i] = wordlist[w]
	}
	return strings.Join(words, " "), nil
}

func decodeShare(mnemonic string) (rawShare, error) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(mnemonic)))
	if len(fields) < minMnemonicWords {
		return rawShare{}, ErrLength
	}
	words := make([]int, len(fields))
	for i, f := range fields {
		idx, ok := wordIndex[f]
		if !ok {
			return rawShare{}, ErrInvalidWord
		}
		words[i] = idx
	}

	bits := wordsToBits(words)
	rs := rawShare{
		id:              readBits(bits, 0, 15),
		extendable:      bits[15] == 1,
		iterationExp:    readBits(bits, 16, 4),
		groupIndex:      readBits(bits, 20, 4),
		groupThreshold:  readBits(bits, 24, 4) + 1,
		groupCount:      readBits(bits, 28, 4) + 1,
		memberIndex:     readBits(bits, 32, 4),
		memberThreshold: readBits(bits, 36, 4) + 1,
	}

	if !rs1024Verify(rs.extendable, words) {
		return rawShare{}, ErrChecksum
	}

	// Value padding: per SLIP-39 the value words carry (10*valueWords) bits;
	// the padding is that count mod 16 and must not exceed 8 bits (and must be
	// zero). This handles every legal secret size (128..256 bits in 16-bit
	// steps), not just 128/256.
	valueBits := bits[40 : len(bits)-30]
	paddingLen := len(valueBits) % 16
	if paddingLen > 8 {
		return rawShare{}, ErrLength
	}
	for i := 0; i < paddingLen; i++ {
		if valueBits[i] != 0 {
			return rawShare{}, ErrPadding
		}
	}
	body := valueBits[paddingLen:]
	nbytes := len(body) / 8
	value := make([]byte, nbytes)
	for i := 0; i < nbytes; i++ {
		value[i] = byte(readBits(body, i*8, 8))
	}
	rs.value = value
	return rs, nil
}

// ---------------------------------------------------------------------------
// Public API.
// ---------------------------------------------------------------------------

// Generate splits masterSecret into a single group of `count` shares requiring
// `threshold` of them to recover. passphrase is the SLIP-39 passphrase (use
// nil/empty for this project). Returns the share mnemonics.
func Generate(masterSecret []byte, threshold, count int, passphrase []byte) ([]string, error) {
	if len(masterSecret) < 16 || len(masterSecret)%2 != 0 {
		return nil, ErrSecretLength
	}
	if threshold < 1 || count < 1 || threshold > count || count > maxShareCount {
		return nil, ErrParams
	}
	const extendable = false
	const iterationExp = 1
	id, err := randID15()
	if err != nil {
		return nil, err
	}
	ems := encrypt(masterSecret, passphrase, iterationExp, id, extendable)

	groupShares, err := splitSecret(1, 1, ems) // single group
	if err != nil {
		return nil, err
	}
	var mnemonics []string
	for _, gs := range groupShares {
		members, err := splitSecret(threshold, count, gs.value)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			s, err := encodeShare(rawShare{
				id:              id,
				extendable:      extendable,
				iterationExp:    iterationExp,
				groupIndex:      int(gs.index),
				groupThreshold:  1,
				groupCount:      1,
				memberIndex:     int(m.index),
				memberThreshold: threshold,
				value:           m.value,
			})
			if err != nil {
				return nil, err
			}
			mnemonics = append(mnemonics, s)
		}
	}
	return mnemonics, nil
}

// Combine recovers the master secret from a set of SLIP-39 mnemonics.
func Combine(mnemonics []string, passphrase []byte) ([]byte, error) {
	if len(mnemonics) == 0 {
		return nil, ErrEmpty
	}
	type group struct {
		memberThreshold int
		shares          []share
	}
	groups := map[int]*group{}
	var ids, exps, gts, gcs = map[int]bool{}, map[int]bool{}, map[int]bool{}, map[int]bool{}
	exts := map[bool]bool{}

	for _, m := range mnemonics {
		s, err := decodeShare(m)
		if err != nil {
			return nil, err
		}
		ids[s.id] = true
		exps[s.iterationExp] = true
		exts[s.extendable] = true
		gts[s.groupThreshold] = true
		gcs[s.groupCount] = true

		g := groups[s.groupIndex]
		if g == nil {
			g = &group{memberThreshold: s.memberThreshold}
			groups[s.groupIndex] = g
		}
		if g.memberThreshold != s.memberThreshold {
			return nil, ErrMemberThreshold
		}
		for _, ex := range g.shares {
			if ex.index == byte(s.memberIndex) {
				return nil, ErrDuplicate
			}
		}
		g.shares = append(g.shares, share{byte(s.memberIndex), s.value})
	}

	if len(ids) != 1 || len(exps) != 1 || len(exts) != 1 {
		return nil, ErrMismatch
	}
	if len(gts) != 1 {
		return nil, ErrGroupThreshold
	}
	if len(gcs) != 1 {
		return nil, ErrGroupCount
	}
	groupThreshold := onlyKey(gts)
	groupCount := onlyKey(gcs)
	if groupThreshold > groupCount {
		return nil, ErrThreshold
	}
	if len(groups) < groupThreshold {
		return nil, ErrInsuffGroups
	}
	if len(groups) != groupThreshold {
		return nil, ErrWrongGroups
	}

	groupShares := make([]share, 0, len(groups))
	for gi, g := range groups {
		if len(g.shares) != g.memberThreshold {
			return nil, ErrWrongMembers
		}
		sec, err := recoverSecret(g.memberThreshold, g.shares)
		if err != nil {
			return nil, err
		}
		groupShares = append(groupShares, share{byte(gi), sec})
	}
	ems, err := recoverSecret(groupThreshold, groupShares)
	if err != nil {
		return nil, err
	}
	return decrypt(ems, passphrase, onlyKey(exps), onlyKey(ids), onlyKeyBool(exts)), nil
}

// ---------------------------------------------------------------------------
// Small helpers.
// ---------------------------------------------------------------------------

func clone(b []byte) []byte { return append([]byte(nil), b...) }

func xor(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func boolBit(b bool) int {
	if b {
		return 1
	}
	return 0
}

func appendBits(bits []int, v, n int) []int {
	for i := n - 1; i >= 0; i-- {
		bits = append(bits, (v>>i)&1)
	}
	return bits
}

func bytesToBits(data []byte) []int {
	bits := make([]int, 0, len(data)*8)
	for _, by := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((by>>i)&1))
		}
	}
	return bits
}

func bitsToWords(bits []int) []int {
	out := make([]int, 0, len(bits)/radixBits)
	for i := 0; i < len(bits); i += radixBits {
		v := 0
		for j := 0; j < radixBits; j++ {
			v = (v << 1) | bits[i+j]
		}
		out = append(out, v)
	}
	return out
}

func wordsToBits(words []int) []int {
	bits := make([]int, 0, len(words)*radixBits)
	for _, w := range words {
		for i := radixBits - 1; i >= 0; i-- {
			bits = append(bits, (w>>i)&1)
		}
	}
	return bits
}

func readBits(bits []int, start, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = (v << 1) | bits[start+i]
	}
	return v
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("slip39: crypto/rand failed: " + err.Error())
	}
	return b
}

func randID15() (int, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return (int(b[0])<<8 | int(b[1])) & 0x7fff, nil
}

func onlyKey(m map[int]bool) int {
	for k := range m {
		return k
	}
	return 0
}

func onlyKeyBool(m map[bool]bool) bool {
	for k := range m {
		return k
	}
	return false
}
