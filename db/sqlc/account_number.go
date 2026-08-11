package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const accountNumberDigits = 10

var accountNumberUpperBound = new(big.Int).Exp(
	big.NewInt(10),
	big.NewInt(accountNumberDigits),
	nil,
)

func generateAccountNumber() (string, error) {
	value, err := rand.Int(rand.Reader, accountNumberUpperBound)
	if err != nil {
		return "", fmt.Errorf("generate account number: %w", err)
	}
	return fmt.Sprintf("%010d", value), nil
}
