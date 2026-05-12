// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package llm implements two complementary paths:
//
// 1) In-process providers — types.go defines LLMHandler and shared types; factory.go
// selects implementations in openai.go, anthropic.go, alibaba.go, coze.go, ollama.go,
// lmstudio.go. These are used when Ling itself calls an LLM (agents, knowledge, etc.).
//
// 2) HTTP relay gateway — gateway_*.go implements the “OpenAPI-compatible HTTP surface”
// for external clients: read the raw request body, forward it to configured llm_channels
// (possibly with failover), stream or buffer the upstream response, and emit usage signals.
// This is not a second “business LLM client”: it is a transparent pipe so customers can
// hit your server like api.openai.com while you inject keys, routing, quotas, and auditing.
// Naming uses “Relay*” to distinguish this path from the handler-based providers above.
package llm
