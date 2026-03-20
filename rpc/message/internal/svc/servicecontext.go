package svc

import (
	"net/http"
	"time"

	"github.com/archyhsh/gochat/pkg/router"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/internal/config"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                config.Config
	SqlConn               sqlx.SqlConn
	Redis                 *redis.Redis
	ConversationModel     model.ConversationModel
	MessageReadModel      model.MessageReadModel
	MessageTemplateModel  model.MessageTemplateModel
	UserConversationModel model.UserConversationModel
	UserRpc               userservice.UserService
	GroupRpc              groupservice.GroupService
	RelationRpc           relationservice.RelationService
	Router                *router.Router
	HttpClient            *http.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	// Robust Redis initialization: uses the main cache node's configuration
	// go-zero's redis.RedisConf internally handles cluster/sentinel if Type is set correctly.
	rdb := redis.MustNewRedis(c.Cache[0].RedisConf)

	return &ServiceContext{
		Config:                c,
		SqlConn:               sqlConn,
		Redis:                 rdb,
		ConversationModel:     model.NewConversationModel(sqlConn, c.Cache),
		MessageReadModel:      model.NewMessageReadModel(sqlConn, c.Cache),
		MessageTemplateModel:  model.NewMessageTemplateModel(sqlConn, c.Cache),
		UserConversationModel: model.NewUserConversationModel(sqlConn, c.Cache),
		UserRpc:               userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		GroupRpc:              groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc)),
		RelationRpc:           relationservice.NewRelationService(zrpc.MustNewClient(c.RelationRpc)),
		Router:                router.NewRouter(rdb, ""),
		HttpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}
