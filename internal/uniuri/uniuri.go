package uniuri

import (
	"crypto/rand"
	"math"
)

const (
	// StdLen is a standard length of uniuri string to achieve ~95 bits of entropy.
	StdLen = 16
	// UUIDLen is a length of uniuri string to achieve ~119 bits of entropy, closest
	// to what can be losslessly converted to UUIDv4 (122 bits).
	UUIDLen = 20
)

// StdChars is a set of standard characters allowed in uniuri string.
var StdChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// New returns a new random string of the standard length, consisting of
// standard characters.
func New() string {
	return NewLenChars(StdLen, StdChars)
}

// NewLen returns a new random string of the provided length, consisting of
// standard characters.
func NewLen(length int) string {
	return NewLenChars(length, StdChars)
}

const (
	// maxBufLen is the maximum length of a temporary buffer for random bytes.
	maxBufLen = 2048

	// minRegenBufLen is the minimum length of temporary buffer for random bytes
	// to fill after the first rand.Read request didn't produce the full result.
	// If the initial buffer is smaller, this value is ignored.
	// Rationale: for performance, assume it's pointless to request fewer bytes from rand.Read.
	minRegenBufLen = 16

	// maxByteValue is the maximum value of a byte (2^8 - 1).
	maxByteValue = 255

	// byteRange is the total number of possible byte values (2^8).
	byteRange = 256
)

// estimatedBufLen returns the estimated number of random bytes to request
// given that byte values greater than maxByte will be rejected.
func estimatedBufLen(need, maxByte int) int {
	return int(math.Ceil(float64(need) * (maxByteValue / float64(maxByte))))
}

// NewLenCharsBytes returns a new random byte slice of the provided length, consisting
// of the provided byte slice of allowed characters (maximum 256).
func NewLenCharsBytes(length int, chars []byte) []byte {
	if length == 0 {
		return nil
	}
	clen := len(chars)
	if clen < 2 || clen > byteRange {
		panic("uniuri: wrong charset length for NewLenChars")
	}

	maxRb := maxByteValue - (byteRange % clen)
	bufLen := estimatedBufLen(length, maxRb)
	if bufLen < length {
		bufLen = length
	}

	if bufLen > maxBufLen {
		bufLen = maxBufLen
	}

	buf := make([]byte, bufLen) // storage for random bytes
	out := make([]byte, length) // storage for result

	var i int // index in out
	for {
		if _, err := rand.Read(buf[:bufLen]); err != nil {
			panic("uniuri: error reading random bytes: " + err.Error())
		}

		for _, rb := range buf[:bufLen] {
			c := int(rb)
			if c > maxRb {
				// Skip this number to avoid modulo bias.
				continue
			}
			out[i] = chars[c%clen]
			i++
			if i == length {
				return out
			}
		}
		// Adjust new requested length, but no smaller than minRegenBufLen.
		bufLen = estimatedBufLen(length-i, maxRb)
		if bufLen < minRegenBufLen && minRegenBufLen < cap(buf) {
			bufLen = minRegenBufLen
		}
		if bufLen > maxBufLen {
			bufLen = maxBufLen
		}
	}
}

// NewLenChars returns a new random string of the provided length, consisting
// of the provided byte slice of allowed characters (maximum 256).
func NewLenChars(length int, chars []byte) string {
	return string(NewLenCharsBytes(length, chars))
}
