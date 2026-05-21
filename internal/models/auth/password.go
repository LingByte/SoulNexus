package auth

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PasswordHashPrefix is the DB storage prefix for password digests (legacy SHA-256 scheme).
// Login JWTs use pkg/utils KeyManager (RS256/ES256), not this format.
const PasswordHashPrefix = "sha256$"

const encryptedPasswordMaxAgeSec = 300

func CheckPassword(user *User, password string) bool {
	if user == nil || user.Password == "" {
		return false
	}
	return user.Password == HashPassword(password)
}

func SetPassword(db *gorm.DB, user *User, password string) error {
	p := HashPassword(password)
	if err := UpdateUserFields(db, user, map[string]any{"Password": p}); err != nil {
		return err
	}
	user.Password = p
	return nil
}

// HashPassword stores a one-way digest. Already-prefixed values are returned unchanged.
func HashPassword(password string) string {
	if password == "" {
		return ""
	}
	if strings.HasPrefix(password, PasswordHashPrefix) {
		return password
	}
	hashVal := sha256.Sum256([]byte(password))
	return PasswordHashPrefix + fmt.Sprintf("%x", hashVal)
}

// PasswordForStorageFromClient normalizes registration payloads from the web client.
// Encrypted transport: passwordHash:encryptedHash:salt:timestamp → sha256$passwordHash.
func PasswordForStorageFromClient(clientPassword string) string {
	parts := strings.Split(clientPassword, ":")
	if len(parts) == 4 && parts[0] != "" {
		if strings.HasPrefix(parts[0], PasswordHashPrefix) {
			return parts[0]
		}
		return PasswordHashPrefix + parts[0]
	}
	return HashPassword(clientPassword)
}

// VerifyEncryptedPassword validates client login payload passwordHash:encryptedHash:salt:timestamp.
func VerifyEncryptedPassword(encryptedPassword, storedPasswordHash string) bool {
	parts := strings.Split(encryptedPassword, ":")
	if len(parts) != 4 {
		return false
	}
	passwordHash, encryptedHash, salt, timestampStr := parts[0], parts[1], parts[2], parts[3]

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}
	originalTimestamp := timestamp
	if timestamp > 9999999999 {
		timestamp = timestamp / 1000
	}
	if time.Now().Unix()-timestamp > encryptedPasswordMaxAgeSec {
		return false
	}

	storedHash := strings.TrimPrefix(storedPasswordHash, PasswordHashPrefix)
	if passwordHash != storedHash {
		return false
	}

	hashInput := fmt.Sprintf("%s%s%d", passwordHash, salt, originalTimestamp)
	hashVal := sha256.Sum256([]byte(hashInput))
	return encryptedHash == fmt.Sprintf("%x", hashVal)
}
