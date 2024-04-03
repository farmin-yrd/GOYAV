package helper

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// MakeHash computes a SHA-256 hash of the given io.ReadSeeker content up to a limit of n bytes.
// It resets the reader to the start before returning.
func MakeHash(reader io.ReadSeeker, n int64) (string, error) {
	if err := resetReader(reader); err != nil {
		return "", fmt.Errorf("failed to reset reader: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(reader, n)); err != nil {
		return "", fmt.Errorf("failed to copy limited data to hash: %w", err)
	}
	if err := resetReader(reader); err != nil {
		return "", fmt.Errorf("failed to reset reader: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// IsValidHash checks if the provided hash string is a valid SHA-256 hash.
func IsValidHash(hash string) bool {
	return len(hash) == 64 // SHA-256 hash is 32 bytes, represented as 64 hex characters
}

// MakeID generates a Base64 encoded MD5 hash. It first processes content from the given io.ReadSeeker
// up to a limit of n bytes, then hashes the provided tag. The function ensures that the reader
// is reset to the start before returning.
func MakeID(reader io.ReadSeeker, n int64, tag string) (string, error) {
	if err := resetReader(reader); err != nil {
		return "", fmt.Errorf("failed to reset reader: %w", err)
	}

	hash := md5.New()
	if _, err := io.Copy(hash, io.LimitReader(reader, n)); err != nil {
		return "", fmt.Errorf("failed to copy limited data to hash: %w", err)
	}
	if _, err := io.Copy(hash, strings.NewReader(tag)); err != nil {
		return "", fmt.Errorf("failed to copy tag to hash: %w", err)
	}

	encodedID := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

	if err := resetReader(reader); err != nil {
		return "", fmt.Errorf("failed to reset reader: %w", err)
	}

	return encodedID, nil
}

// IsValidID checks if the provided ID string is a valid base64 encoded MD5 hash.
func IsValidID(id string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return false
	}
	return len(decoded) == md5.Size // MD5 hash is 16 bytes
}

// resetReader seeks the reader back to the start position.
func resetReader(reader io.ReadSeeker) error {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek reader: %w", err)
	}
	return nil
}
