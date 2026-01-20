package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserConstants(t *testing.T) {
	// Test user signal constants
	assert.Equal(t, "user.login", SigUserLogin, "SigUserLogin should be 'user.login'")
	assert.Equal(t, "user.logout", SigUserLogout, "SigUserLogout should be 'user.logout'")
	assert.Equal(t, "user.create", SigUserCreate, "SigUserCreate should be 'user.create'")
	assert.Equal(t, "user.verifyemail", SigUserVerifyEmail, "SigUserVerifyEmail should be 'user.verifyemail'")
	assert.Equal(t, "user.resetpassword", SigUserResetPassword, "SigUserResetPassword should be 'user.resetpassword'")
	assert.Equal(t, "user.changeemail", SigUserChangeEmail, "SigUserChangeEmail should be 'user.changeemail'")
	assert.Equal(t, "user.changeemaildone", SigUserChangeEmailDone, "SigUserChangeEmailDone should be 'user.changeemaildone'")
}
