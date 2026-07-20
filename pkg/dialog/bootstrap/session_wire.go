package bootstrap

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	"github.com/LingByte/SoulNexus/pkg/logger"
)

func init() {
	session.SetProviderHooks(voiceattach.SessionProviderHooks(logger.Lg))
}
