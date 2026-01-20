package task

import (
	"time"

	"gorm.io/gorm"
)

// StartOfflineChecker starts the user offline checking task
func StartOfflineChecker(db *gorm.DB) {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()

	for range ticker.C {
		checkOfflineUsers(db)
	}
}

// checkOfflineUsers checks user status
func checkOfflineUsers(db *gorm.DB) {
	// Implement checking logic
	// For example: Query users whose last activity time exceeds a certain threshold and mark them as offline
}
