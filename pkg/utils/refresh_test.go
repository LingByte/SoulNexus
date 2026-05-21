package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefreshSignParse(t *testing.T) {
	km := testJWTKeyManager(t)
	p := RefreshPayload{UserID: 7}
	tok, err := SignRefreshTokenWithKey(p, km, 2*time.Hour)
	require.NoError(t, err)

	out, err := ParseRefreshTokenWithKey(tok, km)
	require.NoError(t, err)
	require.Equal(t, uint(7), out.UserID)

	km2 := testJWTKeyManager(t)
	_, err = ParseRefreshTokenWithKey(tok, km2)
	require.Error(t, err)
}
