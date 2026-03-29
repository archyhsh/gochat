package logic

import (
	"context"
	"strconv"

	"fmt"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupMembersLogic {
	return &GetGroupMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupMembersLogic) GetGroupMembers(in *pb.GetGroupMembersRequest) (*pb.GetGroupMembersResponse, error) {
	// Authentication: Extract user_id from metadata
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found")
	}
	userId, _ := strconv.ParseInt(userIdStrs[0], 10, 64)

	// Authorization: Check if the requester is a member of the group
	caller, err := l.svcCtx.GroupMemberModel.FindOneByGroupIdUserId(l.ctx, in.GroupId, userId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.PermissionDenied, "access denied: not a group member")
		}
		return nil, status.Error(codes.Internal, "failed to verify group membership")
	}
	if caller == nil {
		return nil, status.Error(codes.PermissionDenied, "access denied: not a group member")
	}

	// get all group members in a group in group member model
	members, err := l.svcCtx.GroupMemberModel.FindMembersByGroupId(l.ctx, in.GroupId)
	if err != nil {
		l.Errorf("GetGroupMembers failed to query DB: groupID=%d, error=%v", in.GroupId, err)
		return nil, status.Error(codes.Internal, "failed to query group members")
	}

	if len(members) == 0 {
		return &pb.GetGroupMembersResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}

	var uids []int64
	for _, m := range members {
		uids = append(uids, m.UserId)
	}

	userResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &pb.GetUsersByIdsRequest{
		UserIds: uids,
	})

	userMap := make(map[int64]*pb.User)
	if err == nil && userResp != nil {
		for _, u := range userResp.Users {
			userMap[u.Id] = u
		}
	} else {
		l.Errorf("Scheme C: Failed to batch fetch users: %v", err)
	}

	var pbMembers []*pb.GroupMember
	for _, m := range members {
		avatar := ""
		globalNick := ""
		if u, ok := userMap[m.UserId]; ok {
			globalNick = u.Nickname
			avatar = u.Avatar
		}

		displayNick := globalNick
		if m.Nickname != "" {
			if globalNick != "" && m.Nickname != globalNick {
				displayNick = fmt.Sprintf("%s (%s)", m.Nickname, globalNick)
			} else {
				displayNick = m.Nickname
			}
		}

		if displayNick == "" {
			displayNick = fmt.Sprintf("User %d", m.UserId)
		}

		pbMembers = append(pbMembers, &pb.GroupMember{
			UserId:   m.UserId,
			Nickname: displayNick,
			Avatar:   avatar,
			Role:     int32(m.Role),
			JoinedAt: m.JoinedAt.Unix(),
		})
	}

	return &pb.GetGroupMembersResponse{
		Base:    &pb.BaseResponse{Code: 200, Message: "Success"},
		Members: pbMembers,
	}, nil
}
