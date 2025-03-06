package myplugin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	config := &Config{}
	jsonB, err := json.Marshal(config)
	require.NoError(t, err)
	require.NotNil(t, jsonB)
}
