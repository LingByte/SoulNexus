package chat

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSurfaceForChannel(t *testing.T) {
	api := SurfaceForChannel(models.DialogChannelAPI)
	wecom := SurfaceForChannel(models.DialogChannelWeCom)
	if api.SystemAppendix == "" || wecom.SystemAppendix == "" {
		t.Fatal("expected channel appendix")
	}
	if api.SystemAppendix == wecom.SystemAppendix {
		t.Fatal("api and wecom surfaces should differ")
	}
	if !strings.Contains(wecom.SystemAppendix, "企业微信") {
		t.Fatalf("wecom appendix=%q", wecom.SystemAppendix)
	}
}

func TestLoadSkillsAppendixTenantScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:dialog_skills?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.TenantDialogSkill{}); err != nil {
		t.Fatal(err)
	}
	row := &models.TenantDialogSkill{
		TenantID: 1,
		Code:     "polite-clarify",
		Name:     "礼貌追问",
		Body:     "---\nname: x\n---\n\n先确认缺口再作答。\n",
		Enabled:  true,
	}
	if err := models.CreateTenantDialogSkill(db, row); err != nil {
		t.Fatal(err)
	}
	// Other tenant must not leak.
	_ = models.CreateTenantDialogSkill(db, &models.TenantDialogSkill{
		TenantID: 2, Code: "polite-clarify", Name: "other", Body: "secret", Enabled: true,
	})

	out := LoadSkillsAppendix(db, 1, []string{"polite-clarify"})
	if !strings.Contains(out, "【技能: 礼貌追问】") || !strings.Contains(out, "先确认缺口再作答") {
		t.Fatalf("skills appendix=%q", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatal("leaked other tenant skill")
	}
	if got := LoadSkillsAppendix(db, 1, []string{"missing"}); got != "" {
		t.Fatalf("missing skill should be empty, got %q", got)
	}
}
