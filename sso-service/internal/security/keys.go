package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/LingByte/SoulNexus/sso-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func EnsureSigningKey(db *gorm.DB) (*models.SigningKey, error) {
	var key models.SigningKey
	err := db.Where("active = ?", true).First(&key).Error
	if err == nil {
		return &key, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateDER := x509.MarshalPKCS1PrivateKey(privateKey)
	publicDER := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

	key = models.SigningKey{
		KID:        uuid.NewString(),
		Algorithm:  "RS256",
		PrivatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateDER})),
		PublicPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: publicDER})),
		Active:     true,
	}
	if err := db.Create(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}
