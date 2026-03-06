// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/archyhsh/gochat/api/internal/config"
	"github.com/archyhsh/gochat/api/internal/middleware"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/messageservice"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config         config.Config
	AuthMiddleware rest.Middleware
	JwtManager     *auth.JWTManager
	UserRpc        userservice.UserService
	GroupRpc       groupservice.GroupService
	MessageRpc     messageservice.MessageService
	RelationRpc    relationservice.RelationService
}

func NewServiceContext(c config.Config) *ServiceContext {
	jwtManager := auth.NewJWTManager(c.JWT.JwtSecret, c.JWT.ExpireHours)
	return &ServiceContext{
		Config:         c,
		AuthMiddleware: middleware.NewAuthMiddleware(jwtManager).Handle,
		JwtManager:     jwtManager,
		UserRpc:        userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		GroupRpc:       groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc)),
		MessageRpc:     messageservice.NewMessageService(zrpc.MustNewClient(c.MessageRpc)),
		RelationRpc:    relationservice.NewRelationService(zrpc.MustNewClient(c.RelationRpc)),
	}
}
