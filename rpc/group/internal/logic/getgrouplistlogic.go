package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupListLogic {
	return &GetGroupListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupListLogic) GetGroupList(in *pb.GetGroupListRequest) (*pb.GetGroupListResponse, error) {
	UserId := int64(1) // TODO: get user id from context
	groups, err := l.svcCtx.GroupModel.FindGroupsByUserId(l.ctx, UserId)
	if err != nil && model.ErrNotFound != err {
		return nil, status.Error(codes.Internal, "Failed to get group list")
	}
	if model.ErrNotFound == err {
		return &pb.GetGroupListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Groups: []*pb.GroupSummary{}}, nil
	}
	var groupSummaryList []*pb.GroupSummary
	for _, group := range groups {
		groupSummaryList = append(groupSummaryList, &pb.GroupSummary{
			GroupId: group.Id,
			Name:    group.Name,
			Avatar:  group.Avatar,
		})
	}
	return &pb.GetGroupListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Groups: groupSummaryList}, nil
}
