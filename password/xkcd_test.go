package password

import (
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func TestAppendIfMissing_New(t *testing.T) {
	result := AppendIfMissing([]string{"a", "b"}, "c")
	require.Equal(t, []string{"a", "b", "c"}, result)
}

func TestAppendIfMissing_Duplicate(t *testing.T) {
	original := []string{"a", "b", "c"}
	result := AppendIfMissing(original, "b")
	require.Equal(t, original, result)
}

func TestAppendIfMissing_EmptySlice(t *testing.T) {
	result := AppendIfMissing([]string{}, "a")
	require.Equal(t, []string{"a"}, result)
}

func TestStringInSlice_Found(t *testing.T) {
	require.True(t, StringInSlice("b", []string{"a", "b", "c"}))
}

func TestStringInSlice_NotFound(t *testing.T) {
	require.False(t, StringInSlice("d", []string{"a", "b", "c"}))
}

func TestStringInSlice_EmptyList(t *testing.T) {
	require.False(t, StringInSlice("a", []string{}))
}

func TestNewXKCDGenerator_Default(t *testing.T) {
	gen := NewXKCDGenerator()
	require.NotNil(t, gen)
	require.Empty(t, gen.Separator)
}

func TestNewXKCDGenerator_WithSeparator(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	require.Equal(t, "-", gen.Separator)
}

func TestGeneratePassword_WordCount(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(4, false)
	require.NoError(t, err)

	words := strings.Split(pw, "-")
	require.Len(t, words, 4)
}

func TestGeneratePassword_TitleCase(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(5, false)
	require.NoError(t, err)

	words := strings.Split(pw, "-")
	for _, word := range words {
		require.True(t, unicode.IsUpper(rune(word[0])),
			"word %q should start with uppercase", word)
	}
}

func TestGeneratePassword_NoDuplicateWords(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(10, false)
	require.NoError(t, err)

	words := strings.Split(pw, "-")
	seen := make(map[string]bool)
	for _, word := range words {
		require.False(t, seen[word], "duplicate word: %s", word)
		seen[word] = true
	}
}

func TestGeneratePassword_WithDate(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(2, true)
	require.NoError(t, err)

	// Should start with YYYYMMDD-
	matched, err := regexp.MatchString(`^\d{8}-`, pw)
	require.NoError(t, err)
	require.True(t, matched, "password should start with date: %s", pw)
}

func TestGeneratePassword_NoDate(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(2, false)
	require.NoError(t, err)

	// Should NOT start with digits
	matched, err := regexp.MatchString(`^\d{8}`, pw)
	require.NoError(t, err)
	require.False(t, matched, "password should not start with date: %s", pw)
}

func TestGeneratePassword_ZeroWords(t *testing.T) {
	gen := NewXKCDGenerator()
	_, err := gen.GeneratePassword(0, false)
	require.ErrorIs(t, err, ErrTooFewWords)
}

func TestGeneratePassword_SingleWord(t *testing.T) {
	gen := NewXKCDGenerator(WithSeparator("-"))
	pw, err := gen.GeneratePassword(1, false)
	require.NoError(t, err)
	require.NotContains(t, pw, "-")
	require.NotEmpty(t, pw)
}

func TestGeneratePassword_NoSeparator(t *testing.T) {
	gen := NewXKCDGenerator() // no separator
	pw, err := gen.GeneratePassword(3, false)
	require.NoError(t, err)
	require.NotEmpty(t, pw)
	// Words are concatenated without separator — each starts uppercase
	upperCount := 0
	for _, r := range pw {
		if unicode.IsUpper(r) {
			upperCount++
		}
	}
	require.Equal(t, 3, upperCount, "should have 3 title-cased words")
}
