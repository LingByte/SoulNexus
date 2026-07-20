package constants

// Webhook event types delivered to tenant HTTP endpoints.
const (
	WebhookEventCallStarted   = "call.started"
	WebhookEventCallConnected = "call.connected"
	WebhookEventCallEnded     = "call.ended"
	WebhookEventCallTransfer  = "call.transfer"
	WebhookEventReportDaily   = "report.daily"
	WebhookEventReportWeekly  = "report.weekly"
	WebhookEventDialogTurn    = "dialog.turn_completed"
	WebhookEventDialogEnded   = "dialog.session_ended"
)

// AllWebhookEvents is the supported event catalog for validation.
var AllWebhookEvents = []string{
	WebhookEventCallStarted,
	WebhookEventCallConnected,
	WebhookEventCallEnded,
	WebhookEventCallTransfer,
	WebhookEventReportDaily,
	WebhookEventReportWeekly,
	WebhookEventDialogTurn,
	WebhookEventDialogEnded,
}
