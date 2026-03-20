package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type HandleGroupRequestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHandleGroupRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleGroupRequestLogic {
	return &HandleGroupRequestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HandleGroupRequestLogic) HandleGroupRequest(in *pb.HandleGroupRequestRequest) (*pb.HandleGroupRequestResponse, error) {
	// accept or reject a user from joining the group
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	operatorIdStrs := md.Get("user_id")
	if len(operatorIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	operatorId, _ := strconv.ParseInt(operatorIdStrs[0], 10, 64)

	req, err := l.svcCtx.GroupRequestModel.FindOne(l.ctx, in.RequestId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "request not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.Status != 0 {
		return nil, status.Error(codes.FailedPrecondition, "request already handled")
	}

	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, req.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	if group.OwnerId != operatorId {
		return nil, status.Error(codes.PermissionDenied, "only group owner can handle requests")
	}

	var finalMetaVersion int64
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		if in.Accept {
			maxMembers := group.MaxMembers
			if maxMembers == 0 {
				maxMembers = 500
			}
			if group.MemberCount >= maxMembers {
				return status.Error(codes.ResourceExhausted, "group is full")
			}
		}
		req.Status = 2
		if in.Accept {
			req.Status = 1
		}

		updateReqSql := "update `group_request` set `status` = ? where `id` = ?"
		_, err := session.ExecCtx(ctx, updateReqSql, req.Status, req.Id)
		if err != nil {
			return err
		}

		if in.Accept {
			now := time.Now()
			version := now.UnixNano()
			finalMetaVersion = version
			insertMemberSql := "insert into `group_member` (group_id, user_id, role, joined_at, info_version) values (?, ?, ?, ?, ?)"
			_, err = session.ExecCtx(ctx, insertMemberSql, req.GroupId, req.UserId, 1, now, version)
			if err != nil {
				return err
			}
			updateGroupSql := "update `group` set member_count = member_count + 1, meta_version = ? where id = ?"
			_, err = session.ExecCtx(ctx, updateGroupSql, version, group.Id)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if l.svcCtx.Producer != nil {
		action := "reject"
		if in.Accept {
			action = "join"
		}
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    action,
			"group_id":  req.GroupId,
			"user_id":   req.UserId,
			"version":   finalMetaVersion,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		_ = l.svcCtx.Producer.Send(l.ctx, []byte(strconv.FormatInt(req.GroupId, 10)), data)
	}

	return &pb.HandleGroupRequestResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
