// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package chat

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/lingllm/sandbox"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultSkillAssetsRoot = "data/tenant-dialog-skills"

var (
	skillAssetsMu   sync.RWMutex
	skillAssetsRoot = defaultSkillAssetsRoot

	sandboxOnce sync.Once
	sandboxMgr  sandbox.Manager
	sandboxErr  error
)

// SetSkillAssetsRoot overrides on-disk skill bundle root (tests / deploy).
func SetSkillAssetsRoot(dir string) {
	skillAssetsMu.Lock()
	defer skillAssetsMu.Unlock()
	if strings.TrimSpace(dir) == "" {
		skillAssetsRoot = defaultSkillAssetsRoot
		return
	}
	skillAssetsRoot = strings.TrimSpace(dir)
}

func skillAssetsRootDir() string {
	skillAssetsMu.RLock()
	defer skillAssetsMu.RUnlock()
	return skillAssetsRoot
}

// SkillAssetsDir is the absolute-ish work dir for a tenant skill bundle.
func SkillAssetsDir(tenantID uint, code string) string {
	code = models.NormalizeDialogSkillCode(code)
	return filepath.Join(skillAssetsRootDir(), fmt.Sprintf("%d", tenantID), code)
}

func getSandboxManager() (sandbox.Manager, error) {
	sandboxOnce.Do(func() {
		cfg := sandbox.DefaultConfig()
		// Prefer docker when available; fall back to local (python/node on host).
		cfg.Type = sandbox.SandboxTypeDocker
		cfg.FallbackEnabled = true
		sandboxMgr, sandboxErr = sandbox.NewManager(cfg)
	})
	return sandboxMgr, sandboxErr
}

type skillToolRegistrar interface {
	RegisterFunctionTool(name, description string, parameters interface{}, callback providers.LLMFunctionToolCallback)
}

// RegisterBoundSkillTools registers sandbox-backed tools for python/node skills
// listed in agentConfig.dialogSkills. Prompt-only skills stay in the system appendix.
func RegisterBoundSkillTools(db *gorm.DB, tenantID uint, codes []string, llm any, lg *zap.Logger) {
	if db == nil || tenantID == 0 || len(codes) == 0 || llm == nil {
		return
	}
	reg, ok := llm.(skillToolRegistrar)
	if !ok {
		return
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	rows, err := models.GetTenantDialogSkillsByCodes(db, tenantID, codes)
	if err != nil || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		kind := models.NormalizeDialogSkillKind(row.Kind)
		if kind == models.DialogSkillKindPrompt {
			continue
		}
		row := row
		toolName := "run_skill_" + models.SkillToolSuffix(row.Code)
		desc := strings.TrimSpace(row.Description)
		if desc == "" {
			desc = fmt.Sprintf("Execute tenant skill %s (%s) in sandbox", row.Name, kind)
		}
		params := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Optional stdin / JSON argument passed to the skill script",
				},
			},
		}
		reg.RegisterFunctionTool(toolName, desc, params, func(args map[string]interface{}, _ interface{}) (string, error) {
			input, _ := args["input"].(string)
			out, execErr := ExecuteSkillScript(context.Background(), row, input, lg)
			if execErr != nil {
				return "", execErr
			}
			return out, nil
		})
		lg.Info("dialog skill tool registered",
			zap.String("tool", toolName),
			zap.String("code", row.Code),
			zap.String("kind", kind),
		)
	}
}

// ExecuteSkillScript runs a python/node skill via lingllm sandbox.
func ExecuteSkillScript(ctx context.Context, row models.TenantDialogSkill, stdin string, lg *zap.Logger) (string, error) {
	if lg == nil {
		lg = zap.NewNop()
	}
	kind := models.NormalizeDialogSkillKind(row.Kind)
	if kind == models.DialogSkillKindPrompt {
		return "", fmt.Errorf("prompt skill has no script")
	}
	mgr, err := getSandboxManager()
	if err != nil {
		return "", fmt.Errorf("sandbox: %w", err)
	}
	workDir := SkillAssetsDir(row.TenantID, row.Code)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	entry := strings.TrimSpace(row.EntryFile)
	if entry == "" {
		entry = models.DefaultDialogSkillEntry(kind)
	}
	entry = filepath.Clean(entry)
	if strings.HasPrefix(entry, "..") || filepath.IsAbs(entry) {
		return "", fmt.Errorf("invalid entryFile")
	}
	scriptPath := filepath.Join(workDir, entry)
	if strings.TrimSpace(row.ScriptContent) != "" {
		if err := os.WriteFile(scriptPath, []byte(row.ScriptContent), 0o644); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("skill entry not found: %s", entry)
	}

	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
	}
	cfg := &sandbox.ExecuteConfig{
		Script:       scriptPath,
		WorkDir:      workDir,
		Timeout:      timeout,
		Stdin:        stdin,
		AllowNetwork: false,
	}
	res, err := mgr.Execute(ctx, cfg)
	if err != nil && res == nil {
		return "", err
	}
	payload := map[string]any{
		"ok":       res != nil && res.IsSuccess(),
		"exitCode": 0,
		"stdout":   "",
		"stderr":   "",
	}
	if res != nil {
		payload["exitCode"] = res.ExitCode
		payload["stdout"] = res.Stdout
		payload["stderr"] = res.Stderr
		payload["killed"] = res.Killed
		if res.Error != "" {
			payload["error"] = res.Error
		}
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	raw, _ := json.Marshal(payload)
	lg.Info("dialog skill executed",
		zap.String("code", row.Code),
		zap.String("kind", kind),
		zap.Bool("ok", payload["ok"].(bool)),
	)
	return string(raw), nil
}

// ExtractSkillZip unpacks a zip into the skill assets dir (safe path join).
func ExtractSkillZip(tenantID uint, code string, zipPath string) error {
	code = models.NormalizeDialogSkillCode(code)
	dest := SkillAssetsDir(tenantID, code)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(dest, name)
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) && target != dest {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, 8<<20)) // 8MiB per file
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
