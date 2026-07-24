// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Command storeupload uploads files or directories to the configured object store
// (pkg/stores, driven by STORAGE_KIND and related env vars). Prints public URLs
// suitable for Live2D assets, avatars, etc.
//
// Usage:
//
//	go run ./cmd/tools/storeupload -prefix live2d/haru ./path/to/model/
//	go run ./cmd/tools/storeupload -json -prefix assets ./sprite.png
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/stores"
)

type uploadResult struct {
	LocalPath string `json:"localPath"`
	Key       string `json:"key"`
	URL       string `json:"url"`
}

func main() {
	prefix := flag.String("prefix", "uploads", "object key prefix (no leading slash)")
	recursive := flag.Bool("recursive", true, "when path is a directory, upload all files recursively")
	jsonOut := flag.Bool("json", false, "print JSON array of uploads to stdout")
	dryRun := flag.Bool("dry-run", false, "print keys only, do not upload")
	flag.Parse()

	if err := config.Load(); err != nil {
		fatal(err)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "storeupload: pass one or more files or directories to upload\n")
		flag.Usage()
		os.Exit(2)
	}

	store := stores.Default()
	kind := stores.DefaultStoreKind
	var results []uploadResult

	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			fatal(err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			fatal(err)
		}
		if info.IsDir() {
			if !*recursive {
				fatal(fmt.Errorf("directory %q requires -recursive=true", arg))
			}
			err = filepath.WalkDir(abs, func(p string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() {
					return nil
				}
				if shouldSkipFile(p) {
					return nil
				}
				rel, err := filepath.Rel(abs, p)
				if err != nil {
					return err
				}
				key := joinKey(*prefix, rel)
				urlStr := resolvePublicURL(key)
				if *dryRun {
					fmt.Printf("[dry-run] %s -> %s\n", p, key)
					return nil
				}
				if err := uploadFile(store, key, p); err != nil {
					return err
				}
				results = append(results, uploadResult{LocalPath: p, Key: key, URL: urlStr})
				if !*jsonOut {
					fmt.Printf("%s\n  key: %s\n  url: %s\n\n", p, key, urlStr)
				}
				return nil
			})
			if err != nil {
				fatal(err)
			}
			continue
		}

		if shouldSkipFile(abs) {
			continue
		}
		baseName := filepath.Base(abs)
		key := joinKey(*prefix, baseName)
		urlStr := resolvePublicURL(key)
		if *dryRun {
			fmt.Printf("[dry-run] %s -> %s\n", abs, key)
			continue
		}
		if err := uploadFile(store, key, abs); err != nil {
			fatal(err)
		}
		results = append(results, uploadResult{LocalPath: abs, Key: key, URL: urlStr})
		if !*jsonOut {
			fmt.Printf("%s\n  key: %s\n  url: %s\n\n", abs, key, urlStr)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			fatal(err)
		}
	}

	for _, r := range results {
		if strings.HasSuffix(strings.ToLower(r.Key), ".model3.json") {
			fmt.Fprintf(os.Stderr, "hint: set live2dModelUrl in __LingEchoConfig:\n  %q\n", r.URL)
		}
	}

	if !*jsonOut && len(results) > 0 {
		fmt.Fprintf(os.Stderr, "storage kind: %s (%d file(s) uploaded)\n", kind, len(results))
	}
}

func joinKey(prefix, rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		return rel
	}
	return path.Join(prefix, rel)
}

func shouldSkipFile(p string) bool {
	base := filepath.Base(p)
	if base == ".DS_Store" || base == "Thumbs.db" {
		return true
	}
	if strings.HasPrefix(base, ".") {
		return true
	}
	return false
}

func uploadFile(store stores.Store, key, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	// Object stores may reject duplicate keys; upload CLI always replaces.
	_ = store.Delete(key)
	return store.Write(key, f)
}

func resolvePublicURL(key string) string {
	cleanKey := strings.TrimPrefix(strings.TrimSpace(key), "/")
	raw := strings.TrimSpace(stores.Default().PublicURL(cleanKey))
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return raw
	}
	// local：PublicURL 为磁盘路径，与 ginutil.UploadURL 一致时用 SERVER_URL + /uploads/
	parts := strings.Split(cleanKey, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	uploadPath := "/uploads/" + strings.Join(parts, "/")
	if stores.DefaultStoreKind != stores.KindLocal || config.GlobalConfig == nil {
		return uploadPath
	}
	base := strings.TrimRight(strings.TrimSpace(config.GlobalConfig.Server.URL), "/")
	if base == "" {
		return uploadPath
	}
	return base + uploadPath
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "storeupload: %v\n", err)
	os.Exit(1)
}
