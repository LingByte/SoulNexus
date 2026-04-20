package models

import "time"

const (
	AnnouncementStatusDraft     = "draft"
	AnnouncementStatusPublished = "published"
	AnnouncementStatusOffline   = "offline"
)

type Announcement struct {
	BaseModel
	Title     string     `json:"title" gorm:"type:varchar(255);not null;index"`
	Summary   string     `json:"summary" gorm:"type:varchar(500)"`
	Content   string     `json:"content" gorm:"type:text;not null"`
	Status    string     `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	Pinned    bool       `json:"pinned" gorm:"default:false;index"`
	PublishAt *time.Time `json:"publishAt" gorm:"index"`
	ExpireAt  *time.Time `json:"expireAt" gorm:"index"`
}

func (Announcement) TableName() string {
	return "announcements"
}

func NormalizeAnnouncementStatus(status string) string {
	switch status {
	case AnnouncementStatusPublished:
		return AnnouncementStatusPublished
	case AnnouncementStatusOffline:
		return AnnouncementStatusOffline
	default:
		return AnnouncementStatusDraft
	}
}
