// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

var (
	// GlobalKeyManager is the global key manager for JWT signing
	GlobalKeyManager *utils.KeyManager
	stopRotationChan chan struct{}
)

func minKeysToProvision() int {
	n := config.GlobalConfig.JWT.KeepOldKeys
	if n < 2 {
		return 2
	}
	return n
}

// ensureMinimumKeyPairs generates additional key pairs so verification can overlap during rotation
// (JWKS exposes several kids; new tokens use the newest key).
func ensureMinimumKeyPairs(km *utils.KeyManager, keyFile string) error {
	start := km.KeyCount()
	for km.KeyCount() < minKeysToProvision() {
		if _, err := km.GenerateKey(); err != nil {
			return err
		}
		logger.Infof("Provisioned extra signing key (%d/%d)", km.KeyCount(), minKeysToProvision())
	}
	if km.KeyCount() > start {
		if err := km.SaveKeysToFile(keyFile); err != nil {
			return fmt.Errorf("save keys after provisioning: %w", err)
		}
	}
	return nil
}

func publishKeyManager(km *utils.KeyManager) {
	GlobalKeyManager = km
	utils.InstallJWTKeyManager(km)
}

// InitializeKeyManager initializes the global KeyManager with key persistence
func InitializeKeyManager() error {
	keyManager := utils.NewKeyManager(config.GlobalConfig.JWT.Algorithm)
	keyFile := config.GlobalConfig.JWT.KeyFile

	keyDir := filepath.Dir(keyFile)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	loaded := false
	if _, err := os.Stat(keyFile); err == nil {
		if loadErr := keyManager.LoadKeysFromFile(keyFile); loadErr != nil {
			logger.Warnf("failed to load keys from file, will generate new keys: %v", loadErr)
		} else {
			loaded = true
			logger.Infof("Loaded existing keys from %s", keyFile)
			if shouldRotateKey(keyManager) {
				logger.Infof("Key rotation needed, rotating now...")
				if err := rotateKeys(keyManager, keyFile); err != nil {
					logger.Warnf("failed to rotate keys: %v", err)
				}
			}
		}
	}

	if !loaded {
		if _, err := keyManager.GenerateKey(); err != nil {
			return fmt.Errorf("failed to generate new key: %w", err)
		}
		if saveErr := keyManager.SaveKeysToFile(keyFile); saveErr != nil {
			logger.Warnf("failed to save keys to file: %v", saveErr)
		} else {
			logger.Infof("Generated and saved new keys to %s", keyFile)
		}
	}

	if err := ensureMinimumKeyPairs(keyManager, keyFile); err != nil {
		return err
	}

	publishKeyManager(keyManager)
	startKeyRotationChecker(keyManager, keyFile)
	return nil
}

// shouldRotateKey checks if the current key needs rotation
func shouldRotateKey(km *utils.KeyManager) bool {
	currentKey, err := km.GetCurrentKey()
	if err != nil {
		return false
	}

	rotationDays := config.GlobalConfig.JWT.RotationDays
	if rotationDays <= 0 {
		return false
	}

	age := time.Since(currentKey.CreatedAt)
	return age > time.Duration(rotationDays)*24*time.Hour
}

// rotateKeys performs key rotation
func rotateKeys(km *utils.KeyManager, keyFile string) error {
	keepOldKeys := config.GlobalConfig.JWT.KeepOldKeys
	if keepOldKeys <= 0 {
		keepOldKeys = 2
	}

	if err := km.RotateKeys(keepOldKeys); err != nil {
		return fmt.Errorf("failed to rotate keys: %w", err)
	}

	if saveErr := km.SaveKeysToFile(keyFile); saveErr != nil {
		return fmt.Errorf("failed to save rotated keys: %w", saveErr)
	}

	logger.Infof("Keys rotated successfully, keeping last %d keys", keepOldKeys)
	return nil
}

// startKeyRotationChecker starts a background goroutine to check for key rotation
func startKeyRotationChecker(km *utils.KeyManager, keyFile string) {
	stopRotationChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if shouldRotateKey(km) {
					logger.Infof("Scheduled key rotation triggered")
					if err := rotateKeys(km, keyFile); err != nil {
						logger.Errorf("failed to rotate keys: %v", err)
					}
				}
			case <-stopRotationChan:
				logger.Infof("Key rotation checker stopped")
				return
			}
		}
	}()

	logger.Infof("Key rotation checker started (checks every 24 hours)")
}

// StopKeyRotationChecker stops the background key rotation checker
func StopKeyRotationChecker() {
	if stopRotationChan != nil {
		close(stopRotationChan)
	}
}
