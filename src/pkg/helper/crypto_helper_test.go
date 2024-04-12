package helper

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestMakeHashAndID(t *testing.T) {
	testCases := []struct {
		desc     string
		reader   io.Reader
		n        int64
		tag      string
		wantHash bool
		wantID   bool
	}{
		{"empty reader", bytes.NewReader(nil), 10, "tag", true, true},
		{"empty hash", strings.NewReader("hello"), 5, "", true, true},
		{"short reader", strings.NewReader("test"), 10, "tag", true, true},
		{"exact size reader", strings.NewReader("1234567890"), 10, "tag", true, true},
		{"long reader", strings.NewReader("long content"), 4, "tag", true, true},
	}

	cw := NewCryptoWriter()

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			hash, id, err := cw.GenerateHashAndID(tc.tag)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got := IsValidHash(hash); got != tc.wantHash {
				t.Errorf("IsValidHash() = %v, want %v", got, tc.wantHash)
			}
			if got := IsValidID(id); got != tc.wantID {
				t.Errorf("IsValidID() = %v, want %v", got, tc.wantID)
			}
		})
	}
}

func TestIsValidHash(t *testing.T) {
	validHash := strings.Repeat("a", 64)
	invalidHash := "not a hash"

	if !IsValidHash(validHash) {
		t.Errorf("IsValidHash(%s) = false, want true", validHash)
	}
	if IsValidHash(invalidHash) {
		t.Errorf("IsValidHash(%s) = true, want false", invalidHash)
	}
}

func TestIsValidID(t *testing.T) {
	b := make([]byte, md5.Size)
	rand.Read(b) // Generate a random MD5 hash
	validID := base64.RawURLEncoding.EncodeToString(b)

	invalidID := "not an id"

	if !IsValidID(validID) {
		t.Errorf("IsValidID(%s) = false, want true", validID)
	}
	if IsValidID(invalidID) {
		t.Errorf("IsValidID(%s) = true, want false", invalidID)
	}
}
