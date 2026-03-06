package main

import (
	"context"
	"flag"
	"fmt"
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
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	// 1. Initialize Snowflake ID generator (using worker ID 4 for message service)
	if err := snowflake.Init(4); err != nil {
		panic(fmt.Sprintf("Failed to initialize snowflake: %v", err))
	}

	// 2. Start Kafka Consumer for async message persistence and group events
	handler := logic.NewMessageConsumerHandler(ctx)
	consumer, err := kafka.NewConsumer(
		c.Kafka.Brokers,
		c.Kafka.GroupID,
		[]string{c.Kafka.Topics.Message, c.Kafka.Topics.Group},
		handler,
	)
	if err != nil {
		logx.Errorf("Failed to create Kafka consumer: %v", err)
	} else {
		// Run consumer in a separate goroutine
		go func() {
			logx.Infof("Starting Kafka consumer for topics: %s, %s", c.Kafka.Topics.Message, c.Kafka.Topics.Group)
			if err := consumer.Start(context.Background()); err != nil {
				logx.Errorf("Kafka consumer error: %v", err)
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
