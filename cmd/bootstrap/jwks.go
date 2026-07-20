// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"go.uber.org/zap"
)

var (
	// GlobalKeyManager is the global key manager for JWT signing
	GlobalKeyManager *access.KeyManager
	stopRotationChan chan struct{}
)

func minKeysToProvision() int {
	n := config.GlobalConfig.JWT.KeepOldKeys
	if n < pkgconst.DefaultMinJWKSKeys {
		return pkgconst.DefaultMinJWKSKeys
	}
	return n
}

// ensureMinimumKeyPairs generates additional key pairs so verification can overlap during rotation
// (JWKS exposes several kids; new tokens use the newest key).
func ensureMinimumKeyPairs(km *access.KeyManager, keyFile string) error {
	start := km.KeyCount()
	for km.KeyCount() < minKeysToProvision() {
		if _, err := km.GenerateKey(); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("Provisioned extra signing key (%d/%d)", km.KeyCount(), minKeysToProvision()))
	}
	if km.KeyCount() > start {
		if err := km.SaveKeysToFile(keyFile); err != nil {
			return fmt.Errorf("save keys after provisioning: %w", err)
		}
	}
	return nil
}

func publishKeyManager(km *access.KeyManager) {
	GlobalKeyManager = km
	access.InstallJWTKeyManager(km)
}

// InitializeKeyManager initializes the global KeyManager with key persistence
func InitializeKeyManager() error {
	keyManager := access.NewKeyManager(config.GlobalConfig.JWT.Algorithm)
	keyFile := config.GlobalConfig.JWT.KeyFile

	keyDir := filepath.Dir(keyFile)
	if err := os.MkdirAll(keyDir, pkgconst.KeyDirPerm); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	loaded := false
	if _, err := os.Stat(keyFile); err == nil {
		if loadErr := keyManager.LoadKeysFromFile(keyFile); loadErr != nil {
			logger.Warn("failed to load keys from file, will generate new keys:", zap.Error(loadErr))
		} else {
			loaded = true
			logger.Info("Loaded existing keys from", zap.String("keyfile", keyFile))
			if shouldRotateKey(keyManager) {
				logger.Info("Key rotation needed, rotating now...")
				if err := rotateKeys(keyManager, keyFile); err != nil {
					logger.Warn("failed to rotate keys: %v", zap.Error(err))
				}
			}
		}
	}

	if !loaded {
		if _, err := keyManager.GenerateKey(); err != nil {
			return fmt.Errorf("failed to generate new key: %w", err)
		}
		if saveErr := keyManager.SaveKeysToFile(keyFile); saveErr != nil {
			logger.Warn("failed to save keys to file:", zap.Error(saveErr))
		} else {
			logger.Info("Generated and saved new keys to %s", zap.String("keys", keyFile))
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
func shouldRotateKey(km *access.KeyManager) bool {
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
func rotateKeys(km *access.KeyManager, keyFile string) error {
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
func startKeyRotationChecker(km *access.KeyManager, keyFile string) {
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
						logger.Error("failed to rotate keys:", zap.Error(err))
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
