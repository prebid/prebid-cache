package utils

import (
	"github.com/gofrs/uuid"
)

func GenerateRandomId() (string, error) {
	u2, err := uuid.NewV4()
	return u2.String(), err
}
