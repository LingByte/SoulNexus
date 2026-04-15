package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	OAuthClientStatusEnabled  = 1
	OAuthClientStatusDisabled = 2
)

// OAuthClient stores OIDC/OAuth client application registration.
type OAuthClient struct {
	ID           uint64    `json:"id" gorm:"primaryKey;autoIncrement;comment:Primary key"`
	ClientID     string    `json:"clientId" gorm:"size:64;not null;uniqueIndex;comment:Public client id"`
	ClientSecret string    `json:"clientSecret" gorm:"size:64;not null;comment:Confidential client secret"`
	Name         string    `json:"name" gorm:"size:32;not null;comment:Application name"`
	RedirectURI  string    `json:"redirectUri" gorm:"size:255;not null;comment:Allowed redirect uri"`
	Status       int8      `json:"status" gorm:"not null;default:1;comment:1 enabled 2 disabled"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}

func GetOAuthClientByClientID(db *gorm.DB, clientID string) (*OAuthClient, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, errors.New("client_id is required")
	}
	var client OAuthClient
	if err := db.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func GetEnabledOAuthClientByClientID(db *gorm.DB, clientID string) (*OAuthClient, error) {
	client, err := GetOAuthClientByClientID(db, clientID)
	if err != nil {
		return nil, err
	}
	if client.Status != OAuthClientStatusEnabled {
		return nil, errors.New("oauth client is disabled")
	}
	return client, nil
}
