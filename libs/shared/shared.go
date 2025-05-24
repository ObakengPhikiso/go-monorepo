package shared

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

func Version() string {
	return "v0.1.0"
}

// GenerateID returns a random hex string of length 16
func GenerateID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// GetEnv returns the value of the environment variable or fallback if not set
func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Logger is a simple wrapper for log.Println
func Logger(msg string, args ...interface{}) {
	log.Printf(msg, args...)
}
