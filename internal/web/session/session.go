package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// Store is the global session store instance.
var Store *session.Store

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

	return Store.Storage.Set(sessionID, out, exp)
}

// Read reads the session data for the given session ID.
func (s *Data) Read(sessionID string) error {
	byteData, err := Store.Storage.Get(sessionID)
	if err != nil {
		return err
	}

	return json.Unmarshal(byteData, s)
}

// Init initializes the session store with the provided storage backend.
func Init(storage storage.Storage) {
	if storage == nil {
		panic("storage is nil")
	}

	Store = session.New(session.Config{
		Storage: storage,
	})
}

// GenerateSessionID generates a new secure random session ID.
func GenerateSessionID() (string, error) {
	// 32 bytes = 256 bits
	b := make([]byte, 32) //nolint:mnd
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
