package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateGroupNicknameLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateGroupNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGroupNicknameLogic {
	return &UpdateGroupNicknameLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateGroupNicknameLogic) UpdateGroupNickname(in *pb.UpdateGroupNicknameRequest) (*pb.UpdateGroupNicknameResponse, error) {
	err := l.svcCtx.GroupMemberModel.UpdateNickname(l.ctx, in.GroupId, in.UserId, in.Nickname)
	if err != nil {
		l.Errorf("UpdateGroupNickname failed: %v", err)
		return nil, err
	}

	// Send group nickname update event
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_nickname_update",
			"group_id":  in.GroupId,
			"user_id":   in.UserId,
			"nickname":  in.Nickname,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		_ = l.svcCtx.Producer.Send([]byte(strconv.FormatInt(in.GroupId, 10)), data)
	}

	return &pb.UpdateGroupNicknameResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
