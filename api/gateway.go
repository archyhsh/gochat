// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	_ "time/tzdata"

	"github.com/archyhsh/gochat/api/internal/config"
	"github.com/archyhsh/gochat/api/internal/handler"
	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/joho/godotenv"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	_ = godotenv.Load("../.env")

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	// Enable CORS for all routes
	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	// 1. Audit Middleware (Log all incoming requests)
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[AUDIT] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			next(w, r)
		}
	})

	// Debug Route
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/ping",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		},
	})

	// 2. Initialize Service Context
	ctx := svc.NewServiceContext(c)

	// 3. Register Business Handlers (API Routes)
	handler.RegisterHandlers(server, ctx)

	// 4. Static Files Discovery
	staticDir := ""
	targets := []string{"/app/web/static", "web/static", "../web/static", "./web/static"}
	for _, t := range targets {
		if info, err := os.Stat(t); err == nil && info.IsDir() {
			staticDir, _ = filepath.Abs(t)
			break
		}
	}

	if staticDir != "" {
		log.Printf("Serving static files from: %s", staticDir)
		fs := http.FileServer(http.Dir(staticDir))

		// Root
		server.AddRoute(rest.Route{
			Method: http.MethodGet,
			Path:   "/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			},
		})

		// Assets (Standard multi-segment support)
		prefixes := []string{"/static/", "/static/css/", "/static/js/", "/static/img/"}
		for _, prefix := range prefixes {
			server.AddRoute(rest.Route{
				Method:  http.MethodGet,
				Path:    prefix + ":file",
				Handler: http.StripPrefix("/static/", fs).ServeHTTP,
			})
		}
	}

	log.Printf("Starting gateway HTTP server at %s:%d...", c.Host, c.Port)
	server.Start()
}
