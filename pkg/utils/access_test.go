package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testJWTKeyManager(t *testing.T) *KeyManager {
	t.Helper()
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	require.NoError(t, err)
	return km
}

func TestSignParseRoundTrip(t *testing.T) {
	km := testJWTKeyManager(t)
	p := AccessPayload{UserID: 42, Email: "a@b.co", Role: "user"}
	tok, err := SignAccessTokenWithKey(p, km, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	out, err := ParseAccessTokenWithKey(tok, km)
	require.NoError(t, err)
	require.Equal(t, uint(42), out.UserID)
	require.Equal(t, "a@b.co", out.Email)
	require.Equal(t, "user", out.Role)
}

// Snowflake-style IDs exceed JS Number.MAX_SAFE_INTEGER; JWT JSON must round-trip in Go.
func TestSignParseRoundTripSnowflakeUID(t *testing.T) {
	km := testJWTKeyManager(t)
	const id uint = 703132154489475072
	p := AccessPayload{UserID: id, Email: "19511899044@163.com", Role: "user"}
	tok, err := SignAccessTokenWithKey(p, km, time.Hour)
	require.NoError(t, err)

	out, err := ParseAccessTokenWithKey(tok, km)
	require.NoError(t, err)
	require.Equal(t, id, out.UserID)
	require.Equal(t, "19511899044@163.com", out.Email)
}
