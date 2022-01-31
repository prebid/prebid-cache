package utils

import (
	"github.com/gofrs/uuid"
)

// GenerateRandomID generates a "github.com/gofrs/uuid" UUID
func GenerateRandomID() (string, error) {
	u2, err := uuid.NewV4()
	return u2.String(), err
}
