package models

import "gorm.io/gorm"

// agentIconColumnProbe is only used to detect/drop legacy `agents.icon` (Agent model no longer maps it).
type agentIconColumnProbe struct {
	Icon string `gorm:"column:icon"`
}

func (agentIconColumnProbe) TableName() string { return "agents" }

// MigrateAgentsDropIconColumn removes legacy `icon` from `agents` after the field was retired from Agent.
func MigrateAgentsDropIconColumn(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("agents") {
		return nil
	}
	if !db.Migrator().HasColumn(&agentIconColumnProbe{}, "Icon") {
		return nil
	}
	return db.Migrator().DropColumn(&agentIconColumnProbe{}, "Icon")
}
