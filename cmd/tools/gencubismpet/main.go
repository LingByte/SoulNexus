// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Command gencubismpet renders one JS embed template per Cubism character from
// templates/cubism-pet.template.js (see presets below).
//
//	go run ./cmd/tools/gencubismpet
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type preset struct {
	File       string
	Name       string
	StorageKey string
	ModelKey   string // CDN object key under cdn.lingecho.com
	IdleMotion string
	TapMotion  string
	Cubism2    bool
	DocLine    string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	tplPath := filepath.Join(root, "templates", "cubism-pet.template.js")
	tpl, err := os.ReadFile(tplPath)
	if err != nil {
		fatal(err)
	}
	cdn := "https://cdn.lingecho.com"

	presets := []preset{
		{
			File: "haru-greeter.js", Name: "Haru", StorageKey: "cubism-pet-haru-greeter-v1",
			ModelKey: "live2d/haru/haru_greeter_t03.model3.json", IdleMotion: "Idle", TapMotion: "Tap",
			DocLine: "Cubism 4 · pixi-live2d-display 示例 Haru（招呼）",
		},
		{
			File: "haru.js", Name: "Haru", StorageKey: "cubism-pet-haru-v1",
			ModelKey: "live2d/haru/Haru.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · Live2D 官方样本 Haru（example/cubism-samples）",
		},
		{
			File: "hiyori.js", Name: "Hiyori", StorageKey: "cubism-pet-hiyori-v1",
			ModelKey: "live2d/hiyori/Hiyori.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Hiyori",
		},
		{
			File: "mao.js", Name: "Mao", StorageKey: "cubism-pet-mao-v1",
			ModelKey: "live2d/mao/Mao.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Mao",
		},
		{
			File: "mark.js", Name: "Mark", StorageKey: "cubism-pet-mark-v1",
			ModelKey: "live2d/mark/Mark.model3.json", IdleMotion: "Idle", TapMotion: "Idle",
			DocLine: "Cubism 4 · 官方样本 Mark",
		},
		{
			File: "natori.js", Name: "Natori", StorageKey: "cubism-pet-natori-v1",
			ModelKey: "live2d/natori/Natori.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Natori",
		},
		{
			File: "ren.js", Name: "Ren", StorageKey: "cubism-pet-ren-v1",
			ModelKey: "live2d/ren/Ren.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Ren",
		},
		{
			File: "rice.js", Name: "Rice", StorageKey: "cubism-pet-rice-v1",
			ModelKey: "live2d/rice/Rice.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Rice",
		},
		{
			File: "wanko.js", Name: "Wanko", StorageKey: "cubism-pet-wanko-v1",
			ModelKey: "live2d/wanko/Wanko.model3.json", IdleMotion: "Idle", TapMotion: "TapBody",
			DocLine: "Cubism 4 · 官方样本 Wanko",
		},
		{
			File: "shizuku.js", Name: "Shizuku", StorageKey: "cubism-pet-shizuku-v1",
			ModelKey: "live2d/shizuku/shizuku.model.json", IdleMotion: "idle", TapMotion: "tap_body",
			Cubism2: true,
			DocLine: "Cubism 2 · pixi-live2d-display 示例 Shizuku",
		},
	}

	for _, p := range presets {
		out := string(tpl)
		slug := strings.TrimSuffix(p.File, ".js")
		repl := map[string]string{
			"__DOC_LINE__":       p.DocLine,
			"__PET_FILE__":       p.File,
			"__PET_NAME__":       p.Name,
			"__STORAGE_KEY__":    p.StorageKey,
			"__CSS_ID__":         "lingecho-cubism-" + slug + "-css",
			"__MODEL_URL__":      cdn + "/" + p.ModelKey,
			"__IDLE_MOTION__":    p.IdleMotion,
			"__TAP_MOTION__":     p.TapMotion,
			"__LLM_MOTION__":     p.TapMotion,
			"__LIVE2D_RUNTIME__": "cubism2",
		}
		if p.Cubism2 {
			repl["__BUILTIN_LIVE2D_SDK__"] = cdn + "/live2d/sdk/cubism2.min.js"
			repl["__RUNTIME_ASSERT_CUBISM__"] = ""
			repl["__RUNTIME_LOAD_CHAIN__"] = `loadScript(LIVE2D_LEGACY_CDN)
            .then(function () {
                return loadScript(PIXI_CDN);
            })
            .then(function () {
                return loadScript(LIVE2D_CDN);
            })`
		} else {
			repl["__LIVE2D_RUNTIME__"] = "cubism4"
			repl["__BUILTIN_LIVE2D_SDK__"] = cdn + "/live2d/sdk/cubism4.min.js"
			repl["__RUNTIME_ASSERT_CUBISM__"] = `
        if (typeof window.Live2DCubismCore === 'undefined') {
            throw new Error('Live2DCubismCore 未加载');
        }`
			repl["__RUNTIME_LOAD_CHAIN__"] = `loadScript(CUBISM_CORE_CDN)
            .then(function () {
                return loadScript(PIXI_CDN);
            })
            .then(function () {
                return loadScript(LIVE2D_CDN);
            })`
		}
		for k, v := range repl {
			out = strings.ReplaceAll(out, k, v)
		}
		dest := filepath.Join(root, "templates", p.File)
		if err := os.WriteFile(dest, []byte(out), 0644); err != nil {
			fatal(err)
		}
		fmt.Println("wrote", dest)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "gencubismpet: %v\n", err)
	os.Exit(1)
}
