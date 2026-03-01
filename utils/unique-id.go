package utils

import (
	"github.com/rs/xid"
)

// GenerateUniqueID returns a new globally unique identifier string using xid.
func GenerateUniqueID() string {
	return xid.New().String()
}
