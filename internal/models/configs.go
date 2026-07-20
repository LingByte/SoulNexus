package models

import (
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

func isNLUConfigKey(key string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case constants.KEY_NLU_MODEL,
		constants.KEY_NLU_TOKENIZER,
		constants.KEY_NLU_INTENTS_CONFIG,
		constants.KEY_NLU_MIN_CONFIDENCE:
		return true
	default:
		return false
	}
}

func ListPage(db *gorm.DB, page, size int, search string) ([]utils.Config, int64, error) {
	if db == nil {
		return nil, 0, apperror.ErrDBNil
	}
	q := db.Model(&utils.Config{})
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("`key` LIKE ? OR `desc` LIKE ?", like, like)
	}
	return utils.FindPage[utils.Config](q, page, size, "`key` ASC", utils.MaxPageSize200)
}

func GetByID(db *gorm.DB, id uint) (utils.Config, error) {
	var row utils.Config
	if err := db.First(&row, id).Error; err != nil {
		return utils.Config{}, err
	}
	return row, nil
}

type CreateInput struct {
	Key      string
	Desc     string
	Value    string
	Format   string
	Autoload bool
	Public   bool
}

func Create(db *gorm.DB, in CreateInput) (*utils.Config, error) {
	key := strings.ToUpper(strings.TrimSpace(in.Key))
	if key == "" {
		return nil, apperror.ErrConfigKeyRequired
	}
	if in.Format == "" {
		return nil, apperror.ErrConfigFormatInvalid
	}
	var n int64
	if err := db.Model(&utils.Config{}).Where("`key` = ?", key).Count(&n).Error; err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, apperror.ErrConfigKeyExists
	}
	row := &utils.Config{
		Key:      key,
		Desc:     strings.TrimSpace(in.Desc),
		Value:    in.Value,
		Format:   in.Format,
		Autoload: in.Autoload,
		Public:   in.Public,
	}
	if err := db.Create(row).Error; err != nil {
		return nil, err
	}
	utils.PurgeConfigCache(key)
	if in.Autoload {
		_ = utils.GetValue(db, key)
	}
	if isAKSKRoutePolicyKey(key) {
		ReloadAKSKRoutePolicy(db)
	}
	if isNLUConfigKey(key) {
		nlu.ResetEngine()
		nlu.Load(db)
	}
	return row, nil
}

type UpdateInput struct {
	Desc     *string
	Value    *string
	Format   *string
	Autoload *bool
	Public   *bool
}

func Update(db *gorm.DB, id uint, in UpdateInput) (*utils.Config, error) {
	row, err := GetByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrConfigNotFound
		}
		return nil, err
	}
	updates := map[string]any{}
	if in.Desc != nil {
		updates["desc"] = strings.TrimSpace(*in.Desc)
	}
	if in.Value != nil {
		updates["value"] = *in.Value
	}
	if in.Format != nil {
		if *in.Format == "" {
			return nil, apperror.ErrConfigFormatInvalid
		}
		updates["format"] = *in.Format
	}
	if in.Autoload != nil {
		updates["autoload"] = *in.Autoload
	}
	if in.Public != nil {
		updates["public"] = *in.Public
	}
	if len(updates) == 0 {
		return &row, nil
	}
	if err := db.Model(&utils.Config{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	utils.PurgeConfigCache(row.Key)
	after, err := GetByID(db, id)
	if err != nil {
		return nil, err
	}
	if isAKSKRoutePolicyKey(after.Key) {
		ReloadAKSKRoutePolicy(db)
	}
	if isNLUConfigKey(after.Key) {
		nlu.ResetEngine()
		nlu.Load(db)
	}
	return &after, nil
}

func Delete(db *gorm.DB, id uint) error {
	row, err := GetByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrConfigNotFound
		}
		return err
	}
	if err := db.Delete(&utils.Config{}, id).Error; err != nil {
		return err
	}
	utils.PurgeConfigCache(row.Key)
	if isAKSKRoutePolicyKey(row.Key) {
		ReloadAKSKRoutePolicy(db)
	}
	if isNLUConfigKey(row.Key) {
		nlu.ResetEngine()
		nlu.Load(db)
	}
	return nil
}
