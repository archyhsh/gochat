package main

import (
	"flag"
	"fmt"
	_ "time/tzdata"

	"github.com/archyhsh/gochat/rpc/group/internal/config"
	"github.com/archyhsh/gochat/rpc/group/internal/server"
	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/joho/godotenv"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/group.yaml", "the config file")

func main() {
	flag.Parse()

	// Load .env file
	_ = godotenv.Load("../.env")

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterGroupServiceServer(grpcServer, server.NewGroupServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
