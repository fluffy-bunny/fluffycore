package password

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Asking for more distinct words than exist used to loop forever.
func TestGeneratePassword_TooManyWordsReturnsError(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		_, err := NewXKCDGenerator().GeneratePassword(len(xkcdWordList)+1, false)
		done <- err
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrTooManyWords)
	case <-time.After(5 * time.Second):
		t.Fatal("GeneratePassword did not return")
	}
}

func TestGeneratePassword_AllWordsDistinct(t *testing.T) {
	n := len(titledWords())
	pw, err := NewXKCDGenerator(WithSeparator(" ")).GeneratePassword(n, false)
	require.NoError(t, err)
	words := strings.Split(pw, " ")
	require.Len(t, words, n)
	seen := map[string]bool{}
	for _, w := range words {
		require.False(t, seen[w], "duplicate %q", w)
		seen[w] = true
	}
}
