package manager

import (
	"errors"
	"sync"

	"github.com/archyhsh/gochat/rpc/pb"
)

type GrpcConnection struct {
	id        string
	userID    int64
	platform  string
	stream    pb.ChatService_StreamMessagesServer
	closeChan chan struct{}
	closeOnce sync.Once
}

func NewGrpcConnection(id string, userID int64, platform string, stream pb.ChatService_StreamMessagesServer) *GrpcConnection {
	return &GrpcConnection{
		id:        id,
		userID:    userID,
		platform:  platform,
		stream:    stream,
		closeChan: make(chan struct{}),
	}
}

func (c *GrpcConnection) GetID() string       { return c.id }
func (c *GrpcConnection) GetUserID() int64    { return c.userID }
func (c *GrpcConnection) GetPlatform() string { return c.platform }

func (c *GrpcConnection) SendMessage(msg interface{}) error {
	protoMsg, ok := msg.(*pb.OutgoingMessage)
	if !ok {
		return errors.New("invalid message type for gRPC connection, expected *pb.OutgoingMessage")
	}
	return c.stream.Send(protoMsg)
}

func (c *GrpcConnection) Close() error {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
	return nil
}
