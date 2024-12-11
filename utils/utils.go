package utils

import (
	"math/rand"

	"github.com/gofrs/uuid"
)

// GenerateRandomID generates a "github.com/gofrs/uuid" UUID
func GenerateRandomID() (string, error) {
	u2, err := uuid.NewV4()
	return u2.String(), err
}

func RandomPick(pickProbability float64) bool {
	if pickProbability == 0.0 {
		return false
	}
	if pickProbability == 1.0 {
		return true
	}
	return rand.Float64() < pickProbability
}
