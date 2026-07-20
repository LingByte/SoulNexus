package realtime

// ResponseStarter is implemented by realtime agents that can force a reply
// without waiting for additional caller audio (proactive greeting / nudges).
type ResponseStarter interface {
	CreateResponse(hint string) error
}

// InputBufferClearer clears the vendor input audio buffer (optional capability).
type InputBufferClearer interface {
	ClearInputAudio() error
}

// ServerVADEnabler switches the vendor session to server-side VAD (optional).
type ServerVADEnabler interface {
	EnableServerVAD() error
}
