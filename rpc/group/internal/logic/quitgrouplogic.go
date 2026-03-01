package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type QuitGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQuitGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitGroupLogic {
	return &QuitGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QuitGroupLogic) QuitGroup(in *pb.QuitGroupRequest) (*pb.QuitGroupResponse, error) {
	// TODO: Get userID from context
	userID := int64(1)

	err := l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		member, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(ctx, in.GroupId, userID)
		if err != nil {
			return err
		}

		err = l.svcCtx.GroupMemberModel.Delete(ctx, member.Id)
		if err != nil {
			return err
		}

		_, err = session.ExecCtx(ctx, "update `group` set member_count = member_count - 1 where id = ?", in.GroupId)
		return err
	})

	if err != nil {
		l.Errorf("Failed to quit group: %v", err)
		return nil, status.Error(codes.Internal, "failed to quit group")
	}

	if l.svcCtx.Config.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "quit",
			"group_id":  in.GroupId,
			"user_id":   userID,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		_ = l.svcCtx.Config.Producer.Send([]byte(key), data)
	}

	return &pb.QuitGroupResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
