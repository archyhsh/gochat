package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	_ "time/tzdata"

	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/message/internal/config"
	"github.com/archyhsh/gochat/rpc/message/internal/logic"
	"github.com/archyhsh/gochat/rpc/message/internal/server"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/joho/godotenv"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/message.yaml", "the config file")

func main() {
	flag.Parse()

	// Load .env file
	_ = godotenv.Load("../.env")

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	// 1. Initialize Snowflake ID generator (using worker ID 4 for message service)
	if err := snowflake.Init(4); err != nil {
		panic(fmt.Sprintf("Failed to initialize snowflake: %v", err))
	}

	// 2. Start Kafka Consumers
	handler := logic.NewMessageConsumerHandler(ctx)
	hostname, _ := os.Hostname()

	// 2.1 Persistence Consumer (Shared GroupID for Message Topic)
	persistenceConsumer, err := kafka.NewConsumer(
		c.Kafka.Brokers,
		c.Kafka.GroupID, // e.g. "message-rpc-consumer-group"
		[]string{c.Kafka.Topics.Message},
		handler,
	)
	if err == nil {
		go func() {
			logx.Infof("Starting persistence consumer for topic: %s", c.Kafka.Topics.Message)
			if err := persistenceConsumer.Start(context.Background()); err != nil {
				logx.Errorf("Persistence consumer error: %v", err)
			}
		}()
	}

	// 2.2 Broadcast Consumer (Unique GroupID for Cache Invalidation Topics)
	broadcastGroupID := fmt.Sprintf("%s-broadcast-%s", c.Kafka.GroupID, hostname)
	broadcastConsumer, err := kafka.NewConsumer(
		c.Kafka.Brokers,
		broadcastGroupID,
		[]string{c.Kafka.Topics.Group, c.Kafka.Topics.User},
		handler,
	)
	if err == nil {
		go func() {
			logx.Infof("Starting broadcast consumer (ID: %s) for topics: %s, %s",
				broadcastGroupID, c.Kafka.Topics.Group, c.Kafka.Topics.User)
			if err := broadcastConsumer.Start(context.Background()); err != nil {
				logx.Errorf("Broadcast consumer error: %v", err)
			}
		}()
	}

	// 3. Start gRPC Server
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterMessageServiceServer(grpcServer, server.NewMessageServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting message rpc server at %s...\n", c.ListenOn)
	s.Start()
}
