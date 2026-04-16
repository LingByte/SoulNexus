package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Outbound / transfer / media env **key names** (for utils.GetEnv). Not a checklist of required .env
// entries—each caller has defaults when unset. Every name below is referenced from pkg/config/sip.go,
// internal/sipserver, or pkg/sip/session; do not remove without updating those sites.

const (
	EnvSIPTargetNumber      = "SIP_TARGET_NUMBER"
	EnvSIPOutboundHost      = "SIP_OUTBOUND_HOST"
	EnvSIPOutboundPort      = "SIP_OUTBOUND_PORT"
	EnvSIPSignalingAddr     = "SIP_SIGNALING_ADDR"
	EnvSIPOutboundReqURI    = "SIP_OUTBOUND_REQUEST_URI"
	EnvSIPOutboundAutoDial  = "SIP_OUTBOUND_AUTO_DIAL"
	EnvSIPCallerID          = "SIP_CALLER_ID"
	EnvSIPCallerDisplayName = "SIP_CALLER_DISPLAY_NAME"
	EnvSIPDefaultDomain     = "SIP_DEFAULT_DOMAIN"
	EnvSIPDefaultURIPort    = "SIP_DEFAULT_URI_PORT"

	EnvSIPTransferReqURI  = "SIP_TRANSFER_REQUEST_URI"
	EnvSIPTransferSigAddr = "SIP_TRANSFER_SIGNALING_ADDR"
	EnvSIPTransferNumber  = "SIP_TRANSFER_NUMBER"
	EnvSIPTransferHost    = "SIP_TRANSFER_HOST"
	EnvSIPTransferPort    = "SIP_TRANSFER_PORT"

	EnvSIPMediaTXQueueSize = "SIP_MEDIA_TX_QUEUE_SIZE"

	// EnvSIPRegisterPassword: when non-empty, REGISTER (including Expires:0 unregister) must carry the
	// same value in the X-SIP-Register-Password header or the server responds 403 and does not change bindings.
	EnvSIPRegisterPassword = "SIP_PASSWORD"
)
