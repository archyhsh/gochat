package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
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
	members, err := l.svcCtx.GroupMemberModel.FindMembersByGroupId(l.ctx, in.GroupId)
	if err != nil {
		l.Errorf("GetGroupMembers failed: groupID=%d, error=%v", in.GroupId, err)
		return nil, status.Error(codes.Internal, "failed to query group members")
	}

	var pbMembers []*pb.GroupMember
	for _, m := range members {
		pbMembers = append(pbMembers, &pb.GroupMember{
			UserId:   m.UserId,
			Nickname: m.Nickname,
			Role:     int32(m.Role),
			JoinedAt: m.JoinedAt.Unix(),
		})
	}

	return &pb.GetGroupMembersResponse{
		Base:    &pb.BaseResponse{Code: 200, Message: "Success"},
		Members: pbMembers,
	}, nil
}
