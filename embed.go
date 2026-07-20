// Package emailtemplates embeds default HTML mail templates for seeding.
package SoulNexus

import _ "embed"

//go:embed templates/email/welcome.html
var WelcomeHTML string

//go:embed templates/email/verification.html
var VerificationHTML string

//go:embed templates/email/group_invitation.html
var GroupInvitationHTML string

//go:embed templates/email/email_verification.html
var EmailVerificationHTML string

//go:embed templates/email/password_reset.html
var PasswordResetHTML string

//go:embed templates/email/device_verification.html
var DeviceVerificationHTML string

//go:embed templates/email/new_device_login.html
var NewDeviceLoginHTML string

//go:embed templates/email/ai_report_daily.html
var AIReportDailyHTML string

//go:embed templates/email/ai_report_weekly.html
var AIReportWeeklyHTML string

//go:embed templates/embed.js
var LingEchoEmbedJS []byte

//go:embed templates/sprite_idle.png
var LingEchoSpriteIdle []byte

//go:embed templates/sprite_hello.png
var LingEchoSpriteHello []byte

//go:embed templates/icon-lingyu.png
var LingEchoIconLingyu []byte

//go:embed templates/docs.css
var DocsCSS []byte

//go:embed templates/icon-lingyu.png
var DefaultLogoPNG []byte
