package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/zeromicro/go-zero/core/mr"
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
		userConversations, err = l.svcCtx.UserConversationModel.SearchUserConversationsByUserId(l.ctx, userId, in.Keyword)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "Internal database error")
	}

	// 1. Identify missing metadata (Lazy Loading)
	missingUserIds := make([]int64, 0)
	missingGroupIds := make([]int64, 0)
	for _, uc := range userConversations {
		if uc.PeerName == "" {
			if strings.HasPrefix(uc.ConversationId, "group_") {
				missingGroupIds = append(missingGroupIds, uc.PeerId)
			} else {
				missingUserIds = append(missingUserIds, uc.PeerId)
			}
		}
	}

	// 2. Parallel data fetching (Redis & RPC Fallback)
	type metaInfo struct {
		Name   string
		Avatar string
	}
	userMetas := make(map[int64]metaInfo)
	groupMetas := make(map[int64]metaInfo)
	latestSeqs := make(map[string]int64)
	unreadCnts := make(map[string]int64)

	_ = mr.Finish(func() error {
		// RPC Fallback: Only fetch missing User Infos
		if len(missingUserIds) > 0 {
			uResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &pb.GetUsersByIdsRequest{UserIds: missingUserIds})
			if err == nil && uResp != nil {
				for _, u := range uResp.Users {
					userMetas[u.Id] = metaInfo{Name: u.Nickname, Avatar: u.Avatar}
				}
			}
		}
		return nil
	}, func() error {
		// RPC Fallback: Only fetch missing Group Infos
		if len(missingGroupIds) > 0 {
			gResp, err := l.svcCtx.GroupRpc.GetGroupsByIds(l.ctx, &pb.GetGroupsByIdsRequest{GroupIds: missingGroupIds})
			if err == nil && gResp != nil {
				for _, g := range gResp.Groups {
					groupMetas[g.Id] = metaInfo{Name: g.Name, Avatar: g.Avatar}
				}
			}
		}
		return nil
	}, func() error {
		// Batch fetch LatestSequence from Redis (Shared for all)
		keys := make([]string, len(userConversations))
		for i, uc := range userConversations {
			keys[i] = fmt.Sprintf("conv:latest_seq:%s", uc.ConversationId)
		}
		vals, err := l.svcCtx.Redis.Mget(keys...)
		if err == nil {
			for i, v := range vals {
				if v != "" {
					seq, _ := strconv.ParseInt(v, 10, 64)
					latestSeqs[userConversations[i].ConversationId] = seq
				}
			}
		}
		return nil
	}, func() error {
		// Batch fetch Unread Counts from Redis (For private chats)
		privateKeys := make([]string, 0)
		mapping := make([]string, 0)
		for _, uc := range userConversations {
			if !strings.HasPrefix(uc.ConversationId, "group_") {
				key := fmt.Sprintf("unread:cnt:%d:%s", userId, uc.ConversationId)
				privateKeys = append(privateKeys, key)
				mapping = append(mapping, uc.ConversationId)
			}
		}
		if len(privateKeys) > 0 {
			vals, err := l.svcCtx.Redis.Mget(privateKeys...)
			if err == nil {
				for i, v := range vals {
					if v != "" {
						cnt, _ := strconv.ParseInt(v, 10, 64)
						unreadCnts[mapping[i]] = cnt
					}
				}
			}
		}
		return nil
	})

	// 3. Final Assembly using DB Snapshot + Fallback Metas
	var conversations []*pb.ConversationInfo
	for _, uc := range userConversations {
		isGroup := strings.HasPrefix(uc.ConversationId, "group_")

		// Latest Sequence & Unread Logic (Redis First)
		latestSeq := uc.LatestSeq
		if s, ok := latestSeqs[uc.ConversationId]; ok {
			latestSeq = s
		}

		unreadCount := int32(uc.UnreadCount)
		if isGroup {
			if latestSeq > uc.ReadSequence {
				unreadCount = int32(latestSeq - uc.ReadSequence)
			}
		} else {
			if c, ok := unreadCnts[uc.ConversationId]; ok {
				unreadCount = int32(c)
			}
		}

		// Use Redundant Data from DB Snapshot (PeerName/Avatar)
		// Only fallback to RPC metas if DB snapshot is empty
		nickname := uc.PeerName
		avatar := uc.PeerAvatar

		if nickname == "" {
			if isGroup {
				if m, ok := groupMetas[uc.PeerId]; ok {
					nickname = m.Name
					avatar = m.Avatar
				}
			} else {
				if m, ok := userMetas[uc.PeerId]; ok {
					nickname = m.Name
					avatar = m.Avatar
				}
			}
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
			PeerNickname:    nickname,
			PeerAvatar:      avatar,
		})
	}

	return &pb.GetConversationsResponse{
		Base:          &pb.BaseResponse{Code: 200, Message: "Success"},
		Conversations: conversations,
	}, nil
}
