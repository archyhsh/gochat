package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}

	applies, err := l.svcCtx.FriendApplyModel.FindApplyListByToUserId(l.ctx, userId)
	if err != nil && model.ErrNotFound != err {
		return nil, status.Error(codes.Internal, "Failed to get apply list")
	}

	var FriendApplyList []*pb.FriendApply
	for _, a := range applies {
		FriendApplyList = append(FriendApplyList, &pb.FriendApply{
			Id:         a.Id,
			FromUserId: a.FromUserId,
			ToUserId:   a.ToUserId,
			Message:    a.Message,
			Status:     int32(a.Status),
			CreatedAt:  a.CreatedAt.Unix(),
		})
	}
	return &pb.GetApplyListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Applies: FriendApplyList}, nil
}
