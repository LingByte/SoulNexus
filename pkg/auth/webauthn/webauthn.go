// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// WebAuthn / Passkey 服务包装：将 go-webauthn 库封装为 SoulNexus 友好的接口。
//
// 用法（控制器层）：
//   svc, _ := webauthnsvc.New(webauthnsvc.Config{RPID: "example.com", RPDisplayName: "SoulNexus", Origins: []string{"https://example.com"}})
//   options, sessionData, _ := svc.BeginRegistration(user)
//   // 把 sessionData JSON 化保存到 PasskeyChallenge.SessionData，options 返回给客户端。
//   // 客户端完成 navigator.credentials.create 后回传 attestation：
//   cred, _ := svc.FinishRegistration(user, sessionData, attestationJSON)

package webauthnsvc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Config 创建 Service 所需。
type Config struct {
	// RPID 必须等于站点 effective domain，例如 "example.com"。
	RPID string
	// RPDisplayName 在 OS 弹窗里展示。
	RPDisplayName string
	// Origins 完整的 https://host[:port] 列表。
	Origins []string
}

// Service 包装 go-webauthn 实例。
type Service struct {
	wa *webauthn.WebAuthn
}

// New 构造服务。
func New(cfg Config) (*Service, error) {
	rpid := strings.TrimSpace(cfg.RPID)
	if rpid == "" {
		return nil, errors.New("webauthn: RPID required")
	}
	display := strings.TrimSpace(cfg.RPDisplayName)
	if display == "" {
		display = "SoulNexus"
	}
	wcfg := &webauthn.Config{
		RPID:          rpid,
		RPDisplayName: display,
		RPOrigins:     cfg.Origins,
	}
	w, err := webauthn.New(wcfg)
	if err != nil {
		return nil, fmt.Errorf("webauthn: init: %w", err)
	}
	return &Service{wa: w}, nil
}

// User 是 SoulNexus 调用方需要适配的接口（避免直接依赖 internal/models）。
type User interface {
	WebAuthnID() []byte
	WebAuthnName() string
	WebAuthnDisplayName() string
	WebAuthnIcon() string
	WebAuthnCredentials() []webauthn.Credential
}

// BeginRegistration 返回客户端需要的 publicKey options（JSON）与 sessionData（应保存）。
func (s *Service) BeginRegistration(u User) (optionsJSON []byte, sessionData []byte, err error) {
	options, sd, err := s.wa.BeginRegistration(u)
	if err != nil {
		return nil, nil, err
	}
	optBytes, err := json.Marshal(options)
	if err != nil {
		return nil, nil, err
	}
	sdBytes, err := json.Marshal(sd)
	if err != nil {
		return nil, nil, err
	}
	return optBytes, sdBytes, nil
}

// FinishRegistration 校验 attestation，返回 *webauthn.Credential 与 challenge base64url。
func (s *Service) FinishRegistration(u User, sessionData []byte, attestationJSON []byte) (*webauthn.Credential, error) {
	var sd webauthn.SessionData
	if err := json.Unmarshal(sessionData, &sd); err != nil {
		return nil, fmt.Errorf("webauthn: bad session data: %w", err)
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(attestationJSON))
	if err != nil {
		return nil, fmt.Errorf("webauthn: parse attestation: %w", err)
	}
	cred, err := s.wa.CreateCredential(u, sd, parsed)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// BeginLogin 发起认证（discoverable credentials；user 可空）。
func (s *Service) BeginLogin(u User) (optionsJSON []byte, sessionData []byte, err error) {
	options, sd, err := s.wa.BeginLogin(u)
	if err != nil {
		return nil, nil, err
	}
	optBytes, err := json.Marshal(options)
	if err != nil {
		return nil, nil, err
	}
	sdBytes, err := json.Marshal(sd)
	if err != nil {
		return nil, nil, err
	}
	return optBytes, sdBytes, nil
}

// FinishLogin 校验 assertion；返回更新后的 Credential（含新的 SignCount）。
func (s *Service) FinishLogin(u User, sessionData []byte, assertionJSON []byte) (*webauthn.Credential, error) {
	var sd webauthn.SessionData
	if err := json.Unmarshal(sessionData, &sd); err != nil {
		return nil, fmt.Errorf("webauthn: bad session data: %w", err)
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(assertionJSON))
	if err != nil {
		return nil, fmt.Errorf("webauthn: parse assertion: %w", err)
	}
	cred, err := s.wa.ValidateLogin(u, sd, parsed)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// CredentialFromBytes 反序列化数据库中持久化的 webauthn.Credential（JSON 格式）。
func CredentialFromBytes(b []byte) (*webauthn.Credential, error) {
	if len(b) == 0 {
		return nil, errors.New("empty credential blob")
	}
	var c webauthn.Credential
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// CredentialToBytes 序列化为 JSON 以便落库。
func CredentialToBytes(c *webauthn.Credential) ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil credential")
	}
	return json.Marshal(c)
}
