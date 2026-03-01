package claims

import (
	"testing"
	"time"

	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
)

func TestNewClaims_Empty(t *testing.T) {
	c := NewClaims()
	require.NotNil(t, c)
	require.Nil(t, c.Get("anything"))
}

func TestNewClaim_TypeAndValue(t *testing.T) {
	claim := NewClaim("email", "user@example.com")
	require.Equal(t, "email", claim.Type())
	require.Equal(t, "user@example.com", claim.Value())
}

func TestClaims_SetGetDelete(t *testing.T) {
	c := NewClaims()

	err := c.Set("key1", "value1")
	require.NoError(t, err)
	require.Equal(t, "value1", c.Get("key1"))

	err = c.Delete("key1")
	require.NoError(t, err)
	require.Nil(t, c.Get("key1"))
}

func TestClaims_SetOverwrite(t *testing.T) {
	c := NewClaims()

	_ = c.Set("key1", "original")
	_ = c.Set("key1", "updated")
	require.Equal(t, "updated", c.Get("key1"))
}

func TestClaims_Enumeration(t *testing.T) {
	c := NewClaims()
	_ = c.Set("a", 1)
	_ = c.Set("b", 2)
	_ = c.Set("c", 3)

	claims := c.Claims()
	require.Len(t, claims, 3)

	found := make(map[string]interface{})
	for _, claim := range claims {
		found[claim.Type()] = claim.Value()
	}

	require.Equal(t, 1, found["a"])
	require.Equal(t, 2, found["b"])
	require.Equal(t, 3, found["c"])
}

func TestClaims_GetAudience_Valid(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeAud, []string{"aud1", "aud2"})

	aud, err := c.(*Claims).GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string{"aud1", "aud2"}, []string(aud))
}

func TestClaims_GetAudience_WrongType(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeAud, 42)

	_, err := c.(*Claims).GetAudience()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a string array")
}

func TestClaims_GetAudience_Missing(t *testing.T) {
	c := NewClaims()

	_, err := c.(*Claims).GetAudience()
	require.Error(t, err)
}

func TestClaims_GetExpirationTime_Valid(t *testing.T) {
	c := NewClaims()
	expValue := float64(1700000000)
	_ = c.Set(fluffycore_wellknown.ClaimTypeExp, expValue)

	exp, err := c.(*Claims).GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, time.Unix(1700000000, 0), exp.Time)
}

func TestClaims_GetExpirationTime_WrongType(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeExp, "not-a-float")

	_, err := c.(*Claims).GetExpirationTime()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a float64")
}

func TestClaims_GetIssuedAt_Valid(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeIat, float64(1700000000))

	iat, err := c.(*Claims).GetIssuedAt()
	require.NoError(t, err)
	require.Equal(t, time.Unix(1700000000, 0), iat.Time)
}

func TestClaims_GetNotBefore_Valid(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeNbf, float64(1700000000))

	nbf, err := c.(*Claims).GetNotBefore()
	require.NoError(t, err)
	require.Equal(t, time.Unix(1700000000, 0), nbf.Time)
}

func TestClaims_GetIssuer_Valid(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeIss, "https://example.com")

	iss, err := c.(*Claims).GetIssuer()
	require.NoError(t, err)
	require.Equal(t, "https://example.com", iss)
}

func TestClaims_GetIssuer_WrongType(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeIss, 12345)

	_, err := c.(*Claims).GetIssuer()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a string")
}

func TestClaims_GetSubject_Valid(t *testing.T) {
	c := NewClaims()
	_ = c.Set(fluffycore_wellknown.ClaimTypeSub, "user-123")

	sub, err := c.(*Claims).GetSubject()
	require.NoError(t, err)
	require.Equal(t, "user-123", sub)
}

func TestClaims_GetSubject_Missing(t *testing.T) {
	c := NewClaims()

	_, err := c.(*Claims).GetSubject()
	require.Error(t, err)
}

func TestClaims_Valid_AlwaysNil(t *testing.T) {
	c := NewClaims()
	require.NoError(t, c.Valid())
}

func TestClaims_JwtClaims_ReturnsSelf(t *testing.T) {
	c := NewClaims()
	jwtClaims := c.(*Claims).JwtClaims()
	require.NotNil(t, jwtClaims)
	// Setting via the original should be visible via JwtClaims
	_ = c.Set("test", "value")
	require.Equal(t, "value", c.Get("test"))
}

func TestClaims_DeleteNonExistent(t *testing.T) {
	c := NewClaims()
	err := c.Delete("nonexistent")
	require.NoError(t, err)
}
