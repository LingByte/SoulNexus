// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/LingByte/SoulNexus/pkg/stores"
)

// mountMediaFileServer wires an http.FileServer at the LocalStore's URL
// prefix (default /media/) rooted at its on-disk Root. Read-only by
// design — POST/PUT are rejected to keep the surface tiny. Add an auth
// middleware here if recordings ever carry private data; today the
// assumption is that this listener already sits behind your edge proxy
// / auth boundary.
func mountMediaFileServer(r gin.IRoutes, local *stores.LocalStore) {
	prefix := strings.TrimSpace(local.URLPrefix)
	if prefix == "" {
		prefix = stores.DefaultLocalURLPrefix
	}
	prefix = "/" + strings.Trim(prefix, "/") + "/"
	root := strings.TrimSpace(local.Root)
	if root == "" {
		root = stores.UploadDir
	}
	fs := http.StripPrefix(strings.TrimSuffix(prefix, "/"), http.FileServer(http.Dir(root)))
	r.Any(prefix+"*filepath", gin.WrapF(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Cache headers tuned for immutable WAV recordings — same key
		// never gets rewritten because the recorder embeds a timestamp.
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		fs.ServeHTTP(w, r)
	}))
	log.Printf("[http] media file server mounted: %s -> %s", prefix, root)
}
