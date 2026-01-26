package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	epoch          = int64(1704067200000) // 2024-01-01 00:00:00 UTC
	workerIDBits   = uint(10)
	sequenceBits   = uint(12)
	workerIDMax    = int64(-1) ^ (int64(-1) << workerIDBits)
	sequenceMax    = int64(-1) ^ (int64(-1) << sequenceBits)
	workerIDShift  = sequenceBits
	timestampShift = sequenceBits + workerIDBits
)

var (
	ErrInvalidWorkerID = errors.New("worker ID must be between 0 and 1023")
	ErrClockBackward   = errors.New("clock moved backward")
)

// Snowflake 雪花算法 ID 生成器
type Snowflake struct {
	mu        sync.Mutex
	timestamp int64
	workerID  int64
	sequence  int64
}

// New 创建新的雪花算法实例
func New(workerID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > workerIDMax {
		return nil, ErrInvalidWorkerID
	}

	return &Snowflake{
		workerID:  workerID,
		timestamp: 0,
		sequence:  0,
	}, nil
}

// Generate 生成唯一 ID
func (s *Snowflake) Generate() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < s.timestamp {
		return 0, ErrClockBackward
	}

	if now == s.timestamp {
		s.sequence = (s.sequence + 1) & sequenceMax
		if s.sequence == 0 {
			// 等待下一毫秒
			for now <= s.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	id := ((now - epoch) << timestampShift) |
		(s.workerID << workerIDShift) |
		s.sequence

	return id, nil
}

// MustGenerate 生成唯一 ID，失败则 panic
func (s *Snowflake) MustGenerate() int64 {
	id, err := s.Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// ParseID 解析 ID
func ParseID(id int64) (timestamp int64, workerID int64, sequence int64) {
	timestamp = (id >> timestampShift) + epoch
	workerID = (id >> workerIDShift) & workerIDMax
	sequence = id & sequenceMax
	return
}

// 全局实例
var defaultSnowflake *Snowflake

// Init 初始化全局实例
func Init(workerID int64) error {
	sf, err := New(workerID)
	if err != nil {
		return err
	}
	defaultSnowflake = sf
	return nil
}

// NextID 生成下一个 ID
func NextID() (int64, error) {
	if defaultSnowflake == nil {
		return 0, errors.New("snowflake not initialized")
	}
	return defaultSnowflake.Generate()
}

// MustNextID 生成下一个 ID，失败则 panic
func MustNextID() int64 {
	id, err := NextID()
	if err != nil {
		panic(err)
	}
	return id
}
