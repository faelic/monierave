package token

import (
	"crypto/sha256"
	"crypto/subtle"
)

func Hash(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func HashMatches(expected []byte, value string) bool {
	actual := Hash(value)
	return len(expected) == len(actual) &&
		subtle.ConstantTimeCompare(expected, actual) == 1
}
