package logic

import (
	"context"
	"strconv"
	"strings"

	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(in *pb.GetConversationsRequest) (*pb.GetConversationsResponse, error) {
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

	var userConversations []*model.UserConversationWithSeq
	if in.Keyword == "" {
		userConversations, err = l.svcCtx.UserConversationModel.GetUserConversationsByUserId(l.ctx, userId)
	} else {
		userConversations, err = l.svcCtx.UserConversationModel.SearchUserConversationsByUserId(l.ctx, userId)
	}
	if err != nil {
		l.Errorf("Failed to get conversations for user %d: %v", userId, err)
		return nil, status.Error(codes.Internal, "Internal database error")
	}

	// Batch Fetch Info if Keyword is present
	peerNames := make(map[string]string)
	if in.Keyword != "" {
		var uids []int64
		var gids []int64
		for _, uc := range userConversations {
			if strings.HasPrefix(uc.ConversationId, "conv_") {
				uids = append(uids, uc.PeerId)
			} else if strings.HasPrefix(uc.ConversationId, "group_") {
				gids = append(gids, uc.PeerId)
			}
		}

		if len(uids) > 0 {
			md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
			outCtx := metadata.NewOutgoingContext(l.ctx, md)
			relResp, _ := l.svcCtx.RelationRpc.GetFriendList(outCtx, &relationservice.GetFriendListRequest{})
			if relResp != nil {
				for _, f := range relResp.Friends {
					name := f.Nickname
					if f.Remark != "" {
						name = f.Remark
					}
					peerNames["u"+strconv.FormatInt(f.UserId, 10)] = name
				}
			}
			userResp, _ := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &userservice.GetUsersByIdsRequest{UserIds: uids})
			if userResp != nil {
				for _, u := range userResp.Users {
					key := "u" + strconv.FormatInt(u.Id, 10)
					if _, exists := peerNames[key]; !exists {
						peerNames[key] = u.Nickname
					}
				}
			}
		}

		for _, gid := range gids {
			gResp, _ := l.svcCtx.GroupRpc.GetGroupInfo(l.ctx, &groupservice.GetGroupInfoRequest{GroupId: gid})
			if gResp != nil && gResp.Group != nil {
				peerNames["g"+strconv.FormatInt(gid, 10)] = gResp.Group.Name
			}
		}
	}

	var conversations []*pb.ConversationInfo
	keyword := strings.ToLower(in.Keyword)

	for _, uc := range userConversations {
		if in.Keyword != "" {
			name := ""
			if strings.HasPrefix(uc.ConversationId, "conv_") {
				name = peerNames["u"+strconv.FormatInt(uc.PeerId, 10)]
			} else {
				name = peerNames["g"+strconv.FormatInt(uc.PeerId, 10)]
			}
			if !strings.Contains(strings.ToLower(name), keyword) {
				continue
			}
		}

		unreadCount := int32(uc.UnreadCount)
		if uc.LatestSeq > uc.ReadSequence {
			unreadCount = int32(uc.LatestSeq - uc.ReadSequence)
		}

		conversations = append(conversations, &pb.ConversationInfo{
			ConversationId:  uc.ConversationId,
			PeerId:          uc.PeerId,
			UnreadCount:     unreadCount,
			LastMsgId:       uc.GlobalLastMsgId,
			LastMessage:     uc.GlobalLastMsgContent,
			LastMsgType:     int32(uc.GlobalLastMsgType),
			LastSenderId:    uc.GlobalLastSenderId,
			LastMessageTime: uc.GlobalLastMsgTime.UnixMilli(),
			IsTop:           int32(uc.IsTop),
			IsMuted:         int32(uc.IsMuted),
			Version:         uc.Version,
		})
	}

	return &pb.GetConversationsResponse{
		Base:          &pb.BaseResponse{Code: 200, Message: "Success"},
		Conversations: conversations,
	}, nil
}
