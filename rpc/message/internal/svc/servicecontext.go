package svc

import (
	"net/http"
	"time"

	"github.com/archyhsh/gochat/pkg/router"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/internal/config"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                config.Config
	SqlConn               sqlx.SqlConn
	ConversationModel     model.ConversationModel
	MessageReadModel      model.MessageReadModel
	MessageTemplateModel  model.MessageTemplateModel
	UserConversationModel model.UserConversationModel
	UserRpc               userservice.UserService
	GroupRpc              groupservice.GroupService
	Router                *router.Router
	HttpClient            *http.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	// Initialize Router using the first cache node
	rdb := redis.MustNewRedis(c.Cache[0].RedisConf)

	return &ServiceContext{
		Config:                c,
		SqlConn:               sqlConn,
		ConversationModel:     model.NewConversationModel(sqlConn, c.Cache),
		MessageReadModel:      model.NewMessageReadModel(sqlConn, c.Cache),
		MessageTemplateModel:  model.NewMessageTemplateModel(sqlConn, c.Cache),
		UserConversationModel: model.NewUserConversationModel(sqlConn, c.Cache),
		UserRpc:               userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		GroupRpc:              groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc)),
		Router:                router.NewRouter(rdb, ""),
		HttpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}
