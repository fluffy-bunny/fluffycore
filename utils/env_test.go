package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const envA = "ENV_A_42fe9"
const envC = "ENV_C_42fe9"

const originalJsonTemplate = `{"a": "${ENV_A_42fe9}","b": ${ENV_A_42fe9},"c": ${ENV_C_42fe9}}`
const expectedJson = `{"a": "42","b": 42,"c": 43}`

func TestEnvReplace(t *testing.T) {
	t.Setenv(envA, "42")
	t.Setenv(envC, "43")

	// Act
	result := ReplaceEnv(originalJsonTemplate, "${%s}")
	require.Equal(t, expectedJson, result)
}
