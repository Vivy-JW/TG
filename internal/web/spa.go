package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler 返回一个供单页应用用的 handler：先按路径找静态文件，
// 找不到就回落 index.html（浏览器路由接管），index.html 也没有则给引导页。
//
// prefix 是该应用被挂载的路径前缀（如 "/debug/"）。前端构建产物里的资源引用
// 带这个前缀（Vite 的 base 配置），所以取文件前必须剥掉。
//
// 回落 index.html 是必需的：刷新页面或直接粘一个深链接时，浏览器请求的是
// 那个深链接本身，后端得回同一份 HTML 让前端路由接管，而不是 404。
func SPAHandler(fsys fs.FS, prefix, title string) http.Handler {
	if fsys == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeLanding(w, title)
		})
	}
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, prefix)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			rel = "."
		}
		if rel != "." {
			if fi, err := fs.Stat(fsys, rel); err == nil && fi.IsDir() {
				rel = "." // 目录一律走 index.html
			}
		}
		if _, err := fs.Stat(fsys, rel); err != nil {
			if _, err := fs.Stat(fsys, "index.html"); err != nil {
				writeLanding(w, title)
				return
			}
			serveIndex(w, fsys, title)
			return
		}
		if rel == "." {
			// 挂载点本身与它下面的目录都直接给入口 HTML。
			// 交给 FileServer 处理目录会得到一次到 "/" 的 301 —— 前缀被丢掉，
			// 用户会从 /debug/ 被弹到根路径。
			serveIndex(w, fsys, title)
			return
		}
		// FileServer 要的是相对 FS 根的路径，所以重写后再交给它
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + rel
		fileServer.ServeHTTP(w, r2)
	})
}

// serveIndex 直接吐出 index.html（不经过 FileServer，避免它对目录做重定向）。
// title 只是给兜底用的：index.html 读不到时（前端没构建）要回落成引导页，
// 而那条路径上不该把标题丢掉 —— 全新克隆的第一次访问走的就是这里。
func serveIndex(w http.ResponseWriter, fsys fs.FS, title string) {
	b, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		writeLanding(w, title)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 不缓存入口 HTML：它引用的是带内容哈希的资源名，缓存住会让用户拿到旧页面。
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

// writeLanding 前端未构建时的引导页。
//
// go build 不依赖 npm 构建（web/dist 有 .gitkeep 占位），所以后端可以先用起来；
// 这个页面告诉用户还差哪一步。
func writeLanding(w http.ResponseWriter, title string) {
	if title == "" {
		title = "TT"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:system-ui,sans-serif;max-width:44rem;margin:4rem auto;padding:0 1rem;line-height:1.7}
code{background:#f2f2f2;padding:.1rem .3rem;border-radius:3px}</style></head>
<body><h2>%s</h2>
<p>前端尚未构建，所以还没有可视化配置页。构建方式：</p>
<pre><code>cd web &amp;&amp; npm install &amp;&amp; npm run build</code></pre>
<p>构建后重新运行 <code>tt serve</code> 即可。</p>
<p>命令行也能改配置：<code>tt env list</code>、<code>tt config show</code>、<code>tt config validate</code>。</p>
</body></html>`, title, title)
}
