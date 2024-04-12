package helper

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"strings"
)

// cryptoWriter implements the io.Writer interface for cryptographic purposes.
// It writes input data to both SHA-256 and MD5 hashes.
type cryptoWriter struct {
	sha256Hash hash.Hash
	md5Hash    hash.Hash
}

// NewCryptoWriter creates and returns a new instance of CryptoWriter.
// It prepares the writer with both SHA-256 and MD5 hash algorithms.
func NewCryptoWriter() *cryptoWriter {
	return &cryptoWriter{
		sha256Hash: sha256.New(),
		md5Hash:    md5.New(),
	}
}

// Write implements the io.Writer interface. It writes data to both the SHA-256 and MD5 hash functions.
// The function ensures that the same data is written to both hashes to maintain consistency.
// It returns the number of bytes written and any error encountered during the write operation.
func (c *cryptoWriter) Write(data []byte) (int, error) {
	if _, err := c.sha256Hash.Write(data); err != nil {
		return 0, err
	}
	if _, err := c.md5Hash.Write(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// GenerateHashAndID calculates and returns the SHA-256 hash and a base64 URL-safe ID derived from the MD5 hash.
// It processes an additional string input for the MD5 hash, allowing separate control over its content.
// Returns the calculated SHA-256 hash, MD5 ID, and any errors encountered.
func (c *cryptoWriter) GenerateHashAndID(tag string) (hash, ID string, err error) {
	hash = fmt.Sprintf("%x", c.sha256Hash.Sum(nil))
	if _, err := io.Copy(c.md5Hash, strings.NewReader(tag)); err != nil {
		return "", "", fmt.Errorf("failed to generate ID: %v", err)
	}
	ID = base64.RawURLEncoding.EncodeToString(c.md5Hash.Sum(nil))
	return hash, ID, nil
}

// IsValidHash checks if the provided hash string is a valid SHA-256 hash.
func IsValidHash(hash string) bool {
	return len(hash) == 64 // SHA-256 hash is 32 bytes, represented as 64 hex characters
}

// IsValidID checks if the provided ID string is a valid base64 encoded MD5 hash.
func IsValidID(id string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return false
	}
	return len(decoded) == md5.Size // MD5 hash is 16 bytes
}
