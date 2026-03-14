package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupRequestsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupRequestsLogic {
	return &GetGroupRequestsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupRequestsLogic) GetGroupRequests(in *pb.GetGroupRequestsRequest) (*pb.GetGroupRequestsResponse, error) {
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found")
	}
	operatorId, _ := strconv.ParseInt(userIdStrs[0], 10, 64)
	var groupIds []int64
	if in.GroupId > 0 {
		ownerId, err := l.svcCtx.GroupModel.CheckOwner(l.ctx, in.GroupId)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check owner")
		}
		if ownerId != operatorId {
			return nil, status.Error(codes.PermissionDenied, "not the owner")
		}
		groupIds = []int64{in.GroupId}
	} else {
		ownedGroups, err := l.svcCtx.GroupModel.FindGroupsByOwner(l.ctx, operatorId)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to find owned groups")
		}
		for _, g := range ownedGroups {
			groupIds = append(groupIds, g.Id)
		}
	}
	if len(groupIds) == 0 {
		return &pb.GetGroupRequestsResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}

	var allRequests []*model.GroupRequest
	for _, gid := range groupIds {
		requests, err := l.svcCtx.GroupRequestModel.FindPendingByGroupId(l.ctx, gid)
		if err == nil {
			allRequests = append(allRequests, requests...)
		}
	}
	if len(allRequests) == 0 {
		return &pb.GetGroupRequestsResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}
	var uids []int64
	for _, r := range allRequests {
		uids = append(uids, r.UserId)
	}
	userResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &userservice.GetUsersByIdsRequest{
		UserIds: uids,
	})

	userMap := make(map[int64]*pb.User)
	if err == nil && userResp != nil {
		for _, u := range userResp.Users {
			userMap[u.Id] = u
		}
	}
	var pbRequests []*pb.GroupRequest
	for _, r := range allRequests {
		nickname := ""
		avatar := ""
		if u, ok := userMap[r.UserId]; ok {
			nickname = u.Nickname
			avatar = u.Avatar
		}
		pbRequests = append(pbRequests, &pb.GroupRequest{
			Id:        r.Id,
			GroupId:   r.GroupId,
			UserId:    r.UserId,
			Message:   r.Message,
			Status:    int32(r.Status),
			CreatedAt: r.CreatedAt.Unix(),
			Nickname:  nickname,
			Avatar:    avatar,
		})
	}

	return &pb.GetGroupRequestsResponse{
		Base:     &pb.BaseResponse{Code: 200, Message: "Success"},
		Requests: pbRequests,
	}, nil
}
