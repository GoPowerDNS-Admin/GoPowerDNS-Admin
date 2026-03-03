package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// storageBackend is the minimal interface for session storage.
type storageBackend interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, exp time.Duration) error
	Delete(key string) error
}

// store is the global session storage backend.
var store storageBackend

// Data represents the session data structure.
type Data struct {
	User models.User
}

// Write writes the session data for the given session ID with an expiration duration.
func (s *Data) Write(sessionID string, exp time.Duration) error {
	out, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return store.Set(sessionID, out, exp)
}

// Read reads the session data for the given session ID.
func (s *Data) Read(sessionID string) error {
	byteData, err := store.Get(sessionID)
	if err != nil {
		return err
	}

	return json.Unmarshal(byteData, s)
}

// DeleteSession deletes the session with the given session ID from the store.
func DeleteSession(sessionID string) error {
	return store.Delete(sessionID)
}

// Init initializes the session store with the provided storage backend.
func Init(s storageBackend) {
	if s == nil {
		panic("storage is nil")
	}

	store = s
}

// GenerateSessionID generates a new secure random session ID.
func GenerateSessionID() (string, error) {
	// 32 bytes = 256 bits
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
