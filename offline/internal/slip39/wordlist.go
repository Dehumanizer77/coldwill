package slip39

import (
	_ "embed"
	"strings"
)

//go:embed wordlist.txt
var wordlistRaw string

// wordlist holds the 1024 official SLIP-39 words (index = 10-bit value).
var wordlist []string

// wordIndex maps a word back to its 10-bit value.
var wordIndex map[string]int

func init() {
	wordlist = strings.Fields(wordlistRaw)
	if len(wordlist) != 1024 {
		panic("slip39: wordlist must contain exactly 1024 words, got " +
			itoa(len(wordlist)))
	}
	wordIndex = make(map[string]int, len(wordlist))
	for i, w := range wordlist {
		wordIndex[w] = i
	}
}

// itoa is a tiny helper to avoid importing strconv just for a panic message.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
