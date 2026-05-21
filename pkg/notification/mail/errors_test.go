package mail

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsPermanentMailError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("sendcloud: 额度不足"), true},
		{errors.New("sendcloud http 403: forbidden"), true},
		{fmt.Errorf("smtp: authentication failed"), true},
		{errors.New("connection reset by peer"), false},
		{errors.New("sendcloud request: timeout"), false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := isPermanentMailError(tc.err); got != tc.want {
			t.Errorf("isPermanentMailError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
