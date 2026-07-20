package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/intentonnx"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	TenantNluStatusDraft    = "draft"
	TenantNluStatusTraining = "training"
	TenantNluStatusReady    = "ready"
	TenantNluStatusFailed   = "failed"
)

// TenantNluIntentDef is one intent label with optional training samples.
type TenantNluIntentDef struct {
	Name          string   `json:"name"`
	Reply         string   `json:"reply"`
	ReplyVariants []string `json:"replyVariants,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	KeywordBonus  float64  `json:"keywordBonus,omitempty"`
	Samples       []string `json:"samples,omitempty"`
}

// TenantNluSpec is the editable intent configuration for a tenant NLU profile.
type TenantNluSpec struct {
	MinSoftmaxProb    float64              `json:"minSoftmaxProb,omitempty"`
	KeywordLogitBonus float64              `json:"keywordLogitBonus,omitempty"`
	MinTopMargin      float64              `json:"minTopMargin,omitempty"`
	DefaultReply      string               `json:"defaultReply,omitempty"`
	Intents           []TenantNluIntentDef `json:"intents"`
}

// TenantNluModel is a tenant-scoped ONNX intent profile (multiple per tenant).
type TenantNluModel struct {
	common.BaseModel

	TenantID      uint           `json:"tenantId,string" gorm:"index;not null"`
	Name          string         `json:"name" gorm:"size:128;not null"`
	Description   string         `json:"description,omitempty" gorm:"type:text"`
	Status        string         `json:"status" gorm:"size:24;index;not null;default:draft"`
	NumClasses    int            `json:"numClasses" gorm:"not null;default:0"`
	MinConfidence float64        `json:"minConfidence" gorm:"not null;default:0.85"`
	Spec          datatypes.JSON `json:"spec" gorm:"type:json;not null"`
	StorageDir    string         `json:"storageDir,omitempty" gorm:"size:512"`
	TrainError    string         `json:"trainError,omitempty" gorm:"type:text"`
}

func (TenantNluModel) TableName() string {
	return constants.TENANT_NLU_MODEL_TABLE_NAME
}

func DefaultTenantNluSpec() TenantNluSpec {
	raw := intentonnx.DefaultIntentConfigJSON()
	var cfg intentonnx.IntentConfig
	_ = json.Unmarshal(raw, &cfg)
	spec := TenantNluSpec{
		MinSoftmaxProb:    cfg.MinSoftmaxProb,
		KeywordLogitBonus: cfg.KeywordLogitBonus,
		MinTopMargin:      cfg.MinTopMargin,
		DefaultReply:      cfg.DefaultReply,
		Intents:           make([]TenantNluIntentDef, 0, len(cfg.Intents)),
	}
	for _, ent := range cfg.Intents {
		spec.Intents = append(spec.Intents, TenantNluIntentDef{
			Name:          ent.Name,
			Reply:         ent.Reply,
			ReplyVariants: append([]string(nil), ent.ReplyVariants...),
			Keywords:      append([]string(nil), ent.Keywords...),
			KeywordBonus:  ent.KeywordBonus,
		})
	}
	if spec.MinSoftmaxProb <= 0 {
		spec.MinSoftmaxProb = 0.22
	}
	if spec.KeywordLogitBonus <= 0 {
		spec.KeywordLogitBonus = 3.5
	}
	if strings.TrimSpace(spec.DefaultReply) == "" {
		spec.DefaultReply = "抱歉，没能完全确定您的需求。"
	}
	return spec
}

func ParseTenantNluSpec(raw datatypes.JSON) (TenantNluSpec, error) {
	if len(raw) == 0 {
		return TenantNluSpec{}, fmt.Errorf("empty nlu spec")
	}
	var spec TenantNluSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return TenantNluSpec{}, err
	}
	return spec, nil
}

func (s TenantNluSpec) ToIntentConfig() *intentonnx.IntentConfig {
	out := &intentonnx.IntentConfig{
		MinSoftmaxProb:    s.MinSoftmaxProb,
		KeywordLogitBonus: s.KeywordLogitBonus,
		MinTopMargin:      s.MinTopMargin,
		DefaultReply:      s.DefaultReply,
		Intents:           make([]intentonnx.IntentEntry, 0, len(s.Intents)),
	}
	for _, ent := range s.Intents {
		kw := append([]string(nil), ent.Keywords...)
		seen := make(map[string]struct{}, len(kw))
		for _, s := range ent.Samples {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			kw = append(kw, s)
		}
		out.Intents = append(out.Intents, intentonnx.IntentEntry{
			Name:          strings.TrimSpace(ent.Name),
			Reply:         strings.TrimSpace(ent.Reply),
			ReplyVariants: ent.ReplyVariants,
			Keywords:      kw,
			KeywordBonus:  ent.KeywordBonus,
		})
	}
	return out
}

func TenantNluStorageRoot() string {
	return filepath.Join("data", "nlu", "tenants")
}

func TenantNluDir(tenantID, modelID uint) string {
	return filepath.Join(TenantNluStorageRoot(), fmt.Sprintf("%d", tenantID), fmt.Sprintf("%d", modelID))
}

func (m *TenantNluModel) ModelPath() string {
	dir := strings.TrimSpace(m.StorageDir)
	if dir == "" {
		dir = TenantNluDir(m.TenantID, m.ID)
	}
	return filepath.Join(dir, "model.onnx")
}

func (m *TenantNluModel) TokenizerPath() string {
	dir := strings.TrimSpace(m.StorageDir)
	if dir == "" {
		dir = TenantNluDir(m.TenantID, m.ID)
	}
	return filepath.Join(dir, "tokenizer.json")
}

func (m *TenantNluModel) IntentsPath() string {
	dir := strings.TrimSpace(m.StorageDir)
	if dir == "" {
		dir = TenantNluDir(m.TenantID, m.ID)
	}
	return filepath.Join(dir, "intents.json")
}

func (m *TenantNluModel) PrototypesPath() string {
	dir := strings.TrimSpace(m.StorageDir)
	if dir == "" {
		dir = TenantNluDir(m.TenantID, m.ID)
	}
	return filepath.Join(dir, "prototypes.json")
}

func ListTenantNluModels(db *gorm.DB, tenantID uint) ([]TenantNluModel, error) {
	var rows []TenantNluModel
	err := db.Where("tenant_id = ?", tenantID).Order("id DESC").Find(&rows).Error
	return rows, err
}

func GetTenantNluModel(db *gorm.DB, tenantID, id uint) (TenantNluModel, error) {
	var row TenantNluModel
	err := db.Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return row, err
}

func CreateTenantNluModel(db *gorm.DB, row *TenantNluModel) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	return db.Create(row).Error
}

func UpdateTenantNluModel(db *gorm.DB, tenantID, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&TenantNluModel{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func DeleteTenantNluModel(db *gorm.DB, tenantID, id uint) error {
	return db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&TenantNluModel{}).Error
}

func GetTenantNluModelByID(db *gorm.DB, id uint) (TenantNluModel, error) {
	var row TenantNluModel
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

func ListTenantNluModelsAdmin(db *gorm.DB, tenantID uint, page, size int) ([]TenantNluModel, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	q := db.Model(&TenantNluModel{})
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []TenantNluModel
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func CountAssistantsByNluModel(db *gorm.DB, tenantID, modelID uint) (int64, error) {
	var n int64
	err := db.Model(&Assistant{}).Where("tenant_id = ? AND nlu_model_id = ?", tenantID, modelID).Count(&n).Error
	return n, err
}

func RemoveDirBestEffort(dir string) {
	if strings.TrimSpace(dir) == "" {
		return
	}
	_ = os.RemoveAll(dir)
}
