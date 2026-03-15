package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchGroupsLogic {
	return &SearchGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchGroupsLogic) SearchGroups(in *pb.SearchGroupsRequest) (*pb.SearchGroupsResponse, error) {
	if len(in.Keyword) == 0 {
		return &pb.SearchGroupsResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}

	groups, err := l.svcCtx.GroupModel.SearchGroupsByName(l.ctx, in.Keyword)
	if err != nil {
		l.Errorf("SearchGroups failed: keyword=%s, error=%v", in.Keyword, err)
		return nil, status.Error(codes.Internal, "failed to search groups")
	}

	var groupSummaries []*pb.GroupSummary
	for _, g := range groups {
		groupSummaries = append(groupSummaries, &pb.GroupSummary{
			GroupId: g.Id,
			Name:    g.Name,
			Avatar:  g.Avatar,
			OwnerId: g.OwnerId,
		})
	}

	return &pb.SearchGroupsResponse{
		Base:   &pb.BaseResponse{Code: 200, Message: "Success"},
		Groups: groupSummaries,
	}, nil
}
