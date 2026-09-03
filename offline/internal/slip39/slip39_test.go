package slip39

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// TestOfficialVectors runs the canonical SLIP-39 test vectors. The vector file
// format is: [description, [mnemonics...], masterSecretHex, xprv].
// An empty masterSecretHex marks an invalid case that must fail to combine.
// All vectors use the passphrase "TREZOR".
func TestOfficialVectors(t *testing.T) {
	data, err := os.ReadFile("vectors.json")
	if err != nil {
		t.Fatalf("read vectors.json: %v", err)
	}
	var vectors [][]json.RawMessage
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("parse vectors.json: %v", err)
	}

	pass := []byte("TREZOR")
	var passed, failed int
	for _, v := range vectors {
		var desc string
		var mnems []string
		var secretHex string
		_ = json.Unmarshal(v[0], &desc)
		_ = json.Unmarshal(v[1], &mnems)
		_ = json.Unmarshal(v[2], &secretHex)

		got, err := Combine(mnems, pass)
		if secretHex == "" {
			if err == nil {
				t.Errorf("FAIL [%s]: expected error, got secret %x", desc, got)
				failed++
			} else {
				passed++
			}
			continue
		}
		if err != nil {
			t.Errorf("FAIL [%s]: unexpected error: %v", desc, err)
			failed++
			continue
		}
		if hex.EncodeToString(got) != secretHex {
			t.Errorf("FAIL [%s]: got %x want %s", desc, got, secretHex)
			failed++
			continue
		}
		passed++
	}
	t.Logf("official vectors: %d passed, %d failed (of %d)", passed, failed, len(vectors))
}

// TestRoundTrip generates 2-of-3 shares and checks every threshold subset
// recovers the original secret, for both empty and non-empty passphrases.
func TestRoundTrip(t *testing.T) {
	secrets := [][]byte{
		[]byte("0123456789abcdef"),     // 16 bytes (128-bit)
		bytes.Repeat([]byte{0x5A}, 20), // 20 bytes (160-bit; 23-word shares — default key size)
		bytes.Repeat([]byte{0xCD}, 24), // 24 bytes (192-bit; exercises %16 padding path)
		bytes.Repeat([]byte{0xAB}, 32), // 32 bytes (256-bit)
	}
	passphrases := [][]byte{nil, []byte("hunter2")}
	for _, secret := range secrets {
		for _, pass := range passphrases {
			mnems, err := Generate(secret, 2, 3, pass)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if len(mnems) != 3 {
				t.Fatalf("expected 3 mnemonics, got %d", len(mnems))
			}
			subsets := [][]int{{0, 1}, {0, 2}, {1, 2}}
			for _, sub := range subsets {
				in := []string{mnems[sub[0]], mnems[sub[1]]}
				got, err := Combine(in, pass)
				if err != nil {
					t.Fatalf("Combine subset %v: %v", sub, err)
				}
				if !bytes.Equal(got, secret) {
					t.Fatalf("subset %v: got %x want %x", sub, got, secret)
				}
			}
			// A single share must not be enough.
			if _, err := Combine(mnems[:1], pass); err == nil {
				t.Fatalf("single share unexpectedly combined")
			}
		}
	}
}
