package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetApplyListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetApplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApplyListLogic {
	return &GetApplyListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetApplyListLogic) GetApplyList(in *pb.GetApplyListRequest) (*pb.GetApplyListResponse, error) {
	UserId := int64(1) // TODO: get user id from context
	groups, err := l.svcCtx.FriendApplyModel.FindApplyListByToUserId(l.ctx, UserId)
	if err != nil && model.ErrNotFound != err {
		return nil, status.Error(codes.Internal, "Failed to get apply list")
	}
	if model.ErrNotFound == err {
		return &pb.GetApplyListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Applies: []*pb.FriendApply{}}, nil
	}
	var FriendApplyList []*pb.FriendApply
	for _, group := range groups {
		FriendApplyList = append(FriendApplyList, &pb.FriendApply{
			Message:   group.Message,
			CreatedAt: group.CreatedAt.Unix(),
		})
	}
	return &pb.GetApplyListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Applies: FriendApplyList}, nil
}
