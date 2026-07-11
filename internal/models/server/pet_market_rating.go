package svcmodels

import (
	"time"

	"gorm.io/gorm"
)

// PetMarketRating stores one user's score for a marketplace listing.
type PetMarketRating struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ListingID string    `json:"listingId" gorm:"index:idx_pet_market_rating_user,priority:1;size:36;not null"`
	UserID    uint      `json:"userId" gorm:"index:idx_pet_market_rating_user,priority:2;not null"`
	Score     int       `json:"score" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (PetMarketRating) TableName() string { return "pet_market_ratings" }

func UpsertPetMarketRating(db *gorm.DB, listingID string, userID uint, score int) error {
	var row PetMarketRating
	err := db.Where("listing_id = ? AND user_id = ?", listingID, userID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return db.Create(&PetMarketRating{ListingID: listingID, UserID: userID, Score: score}).Error
	}
	if err != nil {
		return err
	}
	row.Score = score
	return db.Save(&row).Error
}

func RecomputeListingRating(db *gorm.DB, listing *PetMarketListing) error {
	var agg struct {
		Avg   float64
		Count int64
	}
	if err := db.Model(&PetMarketRating{}).
		Select("AVG(score) as avg, COUNT(*) as count").
		Where("listing_id = ?", listing.ID).
		Scan(&agg).Error; err != nil {
		return err
	}
	listing.Rating = agg.Avg
	listing.RatingCount = int(agg.Count)
	return db.Model(listing).Updates(map[string]interface{}{
		"rating":       listing.Rating,
		"rating_count": listing.RatingCount,
	}).Error
}

func GetUserPetMarketRating(db *gorm.DB, listingID string, userID uint) (int, bool) {
	var row PetMarketRating
	if err := db.Where("listing_id = ? AND user_id = ?", listingID, userID).First(&row).Error; err != nil {
		return 0, false
	}
	return row.Score, true
}
