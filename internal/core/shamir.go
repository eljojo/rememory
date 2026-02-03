package core

import (
	"fmt"

	vault "github.com/hashicorp/vault/shamir"
)

// Split divides a secret into n shares, requiring k to reconstruct.
// Parameters:
//   - secret: the data to split (e.g., a passphrase)
//   - n: total number of shares to create (2-255)
//   - k: minimum shares needed to reconstruct (2-n)
func Split(secret []byte, n, k int) ([][]byte, error) {
	if err := ValidateShamirParams(n, k); err != nil {
		return nil, err
	}

	shares, err := vault.Split(secret, n, k)
	if err != nil {
		return nil, fmt.Errorf("splitting secret: %w", err)
	}

	return shares, nil
}

// Combine reconstructs the secret from k or more shares.
// Returns an error if fewer than 2 shares are provided.
// Note: If corrupted or wrong shares are provided, this may return
// garbage data without error. Use verification hashes to detect this.
func Combine(shares [][]byte) ([]byte, error) {
	if len(shares) < 2 {
		return nil, fmt.Errorf("need at least 2 shares, got %d", len(shares))
	}

	secret, err := vault.Combine(shares)
	if err != nil {
		return nil, fmt.Errorf("combining shares: %w", err)
	}

	return secret, nil
}

// ValidateShamirParams validates the parameters for Shamir's Secret Sharing.
func ValidateShamirParams(n, k int) error {
	if k < 2 {
		return fmt.Errorf("threshold must be at least 2, got %d", k)
	}
	if k > n {
		return fmt.Errorf("threshold (%d) cannot exceed total shares (%d)", k, n)
	}
	if n > 255 {
		return fmt.Errorf("maximum 255 shares supported, got %d", n)
	}
	return nil
}
