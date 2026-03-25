package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type SaveMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveMessageLogic {
	return &SaveMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SaveMessageLogic) SaveMessage(in *pb.SaveMessageRequest) (*pb.SaveMessageResponse, error) {
	tableName := "message_" + time.UnixMilli(in.Message.Timestamp).Format("200601")
	err := l.svcCtx.MessageTemplateModel.CheckTableExist(l.ctx, tableName)
	if err != nil {
		l.Errorf("Failed to ensure table %s exists: %v", tableName, err)
		return nil, status.Error(codes.Internal, "Internal database error")
	}

	var newSeq int64
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		msgModel := &model.MessageTemplate{
			MsgId:     in.Message.MsgId,
			Content:   in.Message.Content,
			MsgType:   int64(in.Message.MsgType),
			SenderId:  in.Message.SenderId,
			CreatedAt: time.UnixMilli(in.Message.Timestamp),
		}

		convType := int32(1)
		targetId := in.Message.ReceiverId
		if in.Message.GroupId > 0 {
			convType = 2
			targetId = in.Message.GroupId
		}

		seq, err := l.svcCtx.ConversationModel.UpdateSeq(ctx, s, in.Message.ConversationId, convType, targetId, msgModel)
		if err != nil {
			return status.Error(codes.Internal, "fail to update conversation table: "+err.Error())
		}
		newSeq = seq
		msgModel.ConversationId = in.Message.ConversationId
		msgModel.ReceiverId = in.Message.ReceiverId
		msgModel.GroupId = in.Message.GroupId
		msgModel.SequenceId = newSeq
		msgModel.Status = 0

		err = l.svcCtx.MessageTemplateModel.InsertToTable(ctx, s, tableName, msgModel)
		if err != nil {
			return status.Error(codes.Internal, "fail to insert msg: "+err.Error())
		}

		// 获取冗余资料快照 (peer_name, peer_avatar)
		peerName := ""
		peerAvatar := ""
		senderName := ""
		senderAvatar := ""

		if in.Message.GroupId > 0 {
			gResp, err := l.svcCtx.GroupRpc.GetGroupInfo(l.ctx, &pb.GetGroupInfoRequest{GroupId: in.Message.GroupId})
			if err == nil && gResp.Group != nil {
				peerName = gResp.Group.Name
				peerAvatar = gResp.Group.Avatar
			}
		} else {
			// 私聊：预取发送者资料
			sResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &pb.GetUsersByIdsRequest{UserIds: []int64{in.Message.SenderId, in.Message.ReceiverId}})
			if err == nil && sResp != nil {
				for _, u := range sResp.Users {
					if u.Id == in.Message.SenderId {
						senderName = u.Nickname
						senderAvatar = u.Avatar
					}
					if u.Id == in.Message.ReceiverId {
						peerName = u.Nickname
						peerAvatar = u.Avatar
					}
				}
			}
		}

		updatedUsers := make(map[int64]bool)
		if in.Message.SenderId > 0 {
			peerId := in.Message.ReceiverId
			pName := peerName
			pAvatar := peerAvatar
			if in.Message.GroupId > 0 {
				peerId = in.Message.GroupId
				pName = peerName
				pAvatar = peerAvatar
			}
			err = l.svcCtx.UserConversationModel.UpdateNewPrivateMsg(ctx, s, in.Message.SenderId, peerId, pName, pAvatar, in.Message.ConversationId, msgModel, false)
			if err != nil {
				return status.Error(codes.Internal, "fail to init sender bookmark: "+err.Error())
			}
			updatedUsers[in.Message.SenderId] = true
		}
		targets := in.Message.TargetIds
		if in.Message.GroupId == 0 && len(targets) == 0 && in.Message.ReceiverId > 0 {
			targets = []int64{in.Message.ReceiverId}
		}

		for _, tid := range targets {
			if tid <= 0 || updatedUsers[tid] {
				continue
			}

			// Block check for targets (receivers)
			if in.Message.GroupId == 0 && in.Message.SenderId > 0 {
				checkResp, err := l.svcCtx.RelationRpc.CheckFriend(l.ctx, &pb.CheckFriendRequest{
					UserId:   tid,                 // The target receiver
					FriendId: in.Message.SenderId, // The sender
				})
				// If blocked, don't update this user's conversation list/unread count
				if err == nil && checkResp.IsBlocked {
					continue
				}
			}

			incUnread := tid != in.Message.SenderId
			peerId := in.Message.SenderId
			pName := senderName
			pAvatar := senderAvatar

			if in.Message.GroupId > 0 {
				peerId = in.Message.GroupId
				pName = peerName
				pAvatar = peerAvatar
			} else if peerId == 0 {
				// For system messages in private chat (conv_A_B)
				parts := strings.Split(in.Message.ConversationId, "_")
				if len(parts) == 3 && parts[0] == "conv" {
					id1, _ := strconv.ParseInt(parts[1], 10, 64)
					id2, _ := strconv.ParseInt(parts[2], 10, 64)
					if tid == id1 {
						peerId = id2
						pName = peerName
					} else {
						peerId = id1
						pName = senderName // This assumes tid is the 'other' person
					}
				} else {
					peerId = in.Message.ReceiverId
					pName = peerName
				}
			}

			err = l.svcCtx.UserConversationModel.UpdateNewPrivateMsg(ctx, s, tid, peerId, pName, pAvatar, in.Message.ConversationId, msgModel, incUnread)
			if err != nil {
				return status.Error(codes.Internal, "fail to init target bookmark: "+err.Error())
			}
			updatedUsers[tid] = true
		}

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to persist message: "+err.Error())
	}

	// Post-Commit logic: Update Redis for fast access (Cache Pre-warming)
	// 1. Update Latest Sequence for the conversation
	seqKey := fmt.Sprintf("conv:latest_seq:%s", in.Message.ConversationId)
	_ = l.svcCtx.Redis.Setex(seqKey, strconv.FormatInt(newSeq, 10), 3600*24*7)

	// 2. For private chats, increment unread counter in Redis
	if in.Message.GroupId == 0 && in.Message.ReceiverId > 0 {
		unreadKey := fmt.Sprintf("unread:cnt:%d:%s", in.Message.ReceiverId, in.Message.ConversationId)
		_, _ = l.svcCtx.Redis.Incr(unreadKey)
		_ = l.svcCtx.Redis.Expire(unreadKey, 3600*24*7)
	}

	return &pb.SaveMessageResponse{
		Base:     &pb.BaseResponse{Code: 200, Message: "Success"},
		Sequence: newSeq,
	}, nil
}
