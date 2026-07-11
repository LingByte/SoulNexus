package svcmodels

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PetMarketListing is a public pet package on the marketplace (no jsSourceId).
type PetMarketListing struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	MarketID      string    `json:"marketId" gorm:"uniqueIndex:idx_pet_market_public_id;size:64"`
	Name          string    `json:"name" gorm:"index"`
	Description   string    `json:"description" gorm:"type:text"`
	Kind          string    `json:"kind" gorm:"size:20;index"`
	AuthorID      uint      `json:"authorId" gorm:"index"`
	GroupID       uint      `json:"groupId" gorm:"index"`
	PackageMeta   string    `json:"packageMeta" gorm:"type:text"`
	Tags          string    `json:"tags" gorm:"size:500"`
	PreviewEmoji  string    `json:"previewEmoji" gorm:"size:16"`
	Visibility    string    `json:"visibility" gorm:"size:20;default:public"`
	DownloadCount int       `json:"downloadCount" gorm:"default:0"`
	ForkCount     int       `json:"forkCount" gorm:"default:0"`
	Rating        float64   `json:"rating" gorm:"default:0"`
	RatingCount   int       `json:"ratingCount" gorm:"default:0"`
	CreatedAt     time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (PetMarketListing) TableName() string { return "pet_market_listings" }

func CreatePetMarketListing(db *gorm.DB, listing *PetMarketListing) error {
	if listing.ID == "" {
		listing.ID = generateUUID()
	}
	if listing.MarketID == "" {
		listing.MarketID = generateUniquePetMarketID(db, listing.GroupID)
	}
	if listing.Visibility == "" {
		listing.Visibility = "public"
	}
	return db.Create(listing).Error
}

func GetPetMarketListingByMarketID(db *gorm.DB, marketID string) (*PetMarketListing, error) {
	var row PetMarketListing
	err := db.Where("market_id = ?", marketID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ListPublicPetMarketListings(db *gorm.DB, page, limit int, keyword string) ([]PetMarketListing, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	q := db.Model(&PetMarketListing{}).Where("visibility = ?", "public")
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PetMarketListing
	err := q.Order("updated_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func generateUniquePetMarketID(db *gorm.DB, groupID uint) string {
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("mkt_%d_%s", groupID, generateRandomString(10))
		var count int64
		if db.Model(&PetMarketListing{}).Where("market_id = ?", id).Count(&count).Error == nil && count == 0 {
			return id
		}
	}
	return fmt.Sprintf("mkt_%d_%s", groupID, generateUUID())
}
