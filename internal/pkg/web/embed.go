package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var DistFS embed.FS

// SPAHandler 返回 SPA 静态文件处理器
//
// 说明：由 go-zero 的 rest.WithNotFoundHandler 调用，负责：
//  1. 提供前端构建产物（/assets/*.js、*.css、favicon 等）
//  2. 前端路由回退：未命中的路径统一返回 index.html
func SPAHandler() http.Handler {
	distFS, err := fs.Sub(DistFS, "dist")
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// API 与健康检查不在此处理
		if strings.HasPrefix(path, "api/") || path == "health" {
			http.NotFound(w, r)
			return
		}

		if path == "" {
			path = "index.html"
		}

		file, err := distFS.Open(path)
		if err != nil {
			// 文件不存在，回退到 index.html（前端路由）
			path = "index.html"
			file, err = distFS.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		defer file.Close()

		// 设置 Content-Type
		switch {
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(path, ".html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case strings.HasSuffix(path, ".json"):
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		case strings.HasSuffix(path, ".ico"):
			w.Header().Set("Content-Type", "image/x-icon")
		case strings.HasSuffix(path, ".png"):
			w.Header().Set("Content-Type", "image/png")
		}

		// 缓存策略
		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		stat, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, path, stat.ModTime(), file.(io.ReadSeeker))
	})
}
