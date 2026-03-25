// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
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

	// Load .env from common locations
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	} else {
		log.Printf("Success: .env file loaded")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	staticDir, _ := filepath.Abs("../web/static")
	fs := http.FileServer(http.Dir(staticDir))

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		},
	})

	staticPaths := []string{"/static/:file", "/static/css/:file", "/static/js/:file"}
	for _, p := range staticPaths {
		server.AddRoute(rest.Route{
			Method: http.MethodGet,
			Path:   p,
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.StripPrefix("/static/", fs).ServeHTTP(w, r)
			},
		})
	}

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting gateway HTTP server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
