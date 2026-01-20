package constants

import "errors"

var (
	ErrSessionIsShutdown  = errors.New("session is shutdown")
	ErrSessionIsRunning   = errors.New("session is running")
	ErrNotInputTransport  = errors.New("not input transport")
	ErrNotOutputTransport = errors.New("not output transport")
	ErrTransportIsClosed  = errors.New("transport is closed")
	ErrInvalidFrameType   = errors.New("invalid frame type")
	ErrCodecNotSupported  = errors.New("codec not supported")
)
