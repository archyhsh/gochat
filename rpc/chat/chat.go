package main

import (
	"flag"
	"fmt"

	"github.com/archyhsh/gochat/internal/gateway/connection"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/rpc/chat/internal/config"
	"github.com/archyhsh/gochat/rpc/chat/internal/server"
	"github.com/archyhsh/gochat/rpc/chat/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/chat.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Initialize dependencies for standalone RPC server (for testing)
	log := logger.New("info", "console")
	manager := connection.NewManager(log)
	manager.Start()

	ctx := svc.NewServiceContext(c, manager)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterChatServiceServer(grpcServer, server.NewChatServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
