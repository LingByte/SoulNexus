package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefreshSignParse(t *testing.T) {
	secret := "another-secret-at-least-8-chars"
	p := RefreshPayload{UserID: 7}
	tok, err := SignRefreshToken(p, secret, 2*time.Hour)
	require.NoError(t, err)

	out, err := ParseRefreshToken(tok, secret)
	require.NoError(t, err)
	require.Equal(t, uint(7), out.UserID)

	_, err = ParseRefreshToken(tok, "wrong-secret-at-least-8")
	require.Error(t, err)
}
