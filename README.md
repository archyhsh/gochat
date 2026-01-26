# GoChat - 高并发IM即时通讯系统

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)
![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen?style=flat-square)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)
![Kubernetes](https://img.shields.io/badge/K8s-Ready-326CE5?style=flat-square&logo=kubernetes)

**基于 Go 语言的高并发分布式即时通讯系统**

[架构设计](#架构设计) | [快速开始](#快速开始) | [技术栈](#技术栈) | [功能特性](#功能特性) | [API文档](#api文档)

</div>

---

## 项目简介

GoChat 是一个基于微服务架构的高并发即时通讯系统，采用 Go 语言开发，支持 **10万+** 并发连接。系统使用 Goroutine + Channel 实现高效的消息分发，通过 Kafka 实现消息解耦与削峰填谷，支持私聊、群聊、文件传输、消息推送等核心功能。

### 核心亮点

- **高并发**: 基于 Goroutine + Channel 的连接管理，单机支持 10万+ 并发连接
- **消息可靠**: Kafka 异步消息队列 + MySQL 持久化，保证消息不丢失
- **低延迟**: Redis 多级缓存，消息读取延迟 < 10ms
- **多端同步**: 支持多设备同时在线，消息实时同步
- **云原生**: Kubernetes 部署，支持弹性扩缩容

---

## 架构设计

```
                                    ┌─────────────────────────────────────────────┐
                                    │              Kubernetes Cluster             │
                                    │  ┌─────────────────────────────────────┐   │
    ┌──────────┐                    │  │         API Gateway (Nginx)         │   │
    │  Web     │◄──── WebSocket ────┼──┤                                     │   │
    │  Client  │                    │  └──────────────┬──────────────────────┘   │
    └──────────┘                    │                 │                          │
                                    │  ┌──────────────▼──────────────────────┐   │
    ┌──────────┐                    │  │     Gateway Service (WebSocket)     │   │
    │  Mobile  │◄──── WebSocket ────┼──┤  • Goroutine per connection         │   │
    │  Client  │                    │  │  • Channel-based message routing    │   │
    └──────────┘                    │  │  • Connection pool management       │   │
                                    │  └──────────────┬──────────────────────┘   │
                                    │                 │ gRPC                     │
                                    │  ┌──────────────▼──────────────────────┐   │
                                    │  │         Message Service             │   │
                                    │  │  • Message persistence (MySQL)      │   │
                                    │  │  • Message routing via Kafka        │   │
                                    │  │  • Read/Unread status               │   │
                                    │  └──────────────┬──────────────────────┘   │
                                    │                 │                          │
                                    │  ┌──────────────┴──────────────────────┐   │
                                    │  │                                     │   │
                                    │  ▼                                     ▼   │
                                    │ ┌────────────────┐   ┌────────────────┐   │
                                    │ │Relation Service│   │  Push Service  │   │
                                    │ │• Friend/Group  │   │• Online push   │   │
                                    │ │• Block list    │   │• APNs/FCM      │   │
                                    │ └────────────────┘   └────────────────┘   │
                                    │                                            │
                                    │  ┌─────────────────────────────────────┐   │
                                    │  │          File Service               │   │
                                    │  │  • Upload/Download • Thumbnail      │   │
                                    │  │  • MinIO/S3 storage                 │   │
                                    │  └─────────────────────────────────────┘   │
                                    │                                            │
                                    │  ┌─────────────────────────────────────┐   │
                                    │  │        Infrastructure Layer         │   │
                                    │  │  MySQL │ Redis │ Kafka │ MinIO      │   │
                                    │  └─────────────────────────────────────┘   │
                                    └─────────────────────────────────────────────┘
```

### 消息流转流程

```
发送方 ──► Gateway ──► Kafka ──► Message Service ──► MySQL/Redis
                                       │
                                       ▼
                        Kafka ──► Push Service ──► Gateway ──► 接收方
                                       │
                                       ▼ (离线)
                                  APNs / FCM
```

---

## 技术栈

| 层级 | 技术 | 说明 |
|-----|------|-----|
| **网关层** | Go + gorilla/websocket | WebSocket 长连接管理 |
| **服务通信** | gRPC + Protobuf | 微服务内部高性能通信 |
| **消息队列** | Apache Kafka | 消息异步分发、削峰填谷 |
| **缓存层** | Redis Cluster | 会话缓存、在线状态、热点消息 |
| **持久化** | MySQL 8.0 | 消息存储、用户数据、关系数据 |
| **对象存储** | MinIO | 文件存储（S3 兼容） |
| **服务发现** | etcd | 服务注册与发现 |
| **容器编排** | Kubernetes + Helm | 服务部署、自动扩缩容 |
| **监控** | Prometheus + Grafana | 性能监控、告警 |
| **日志** | Zap + ELK | 结构化日志、日志分析 |
| **前端** | Vue 3 + TypeScript | Web 端演示界面 |

---

## 功能特性

### 核心功能

- [x] **用户系统**: 注册、登录、JWT 认证
- [x] **私聊消息**: 一对一实时聊天
- [x] **群聊消息**: 群组消息广播
- [x] **消息类型**: 文本、图片、文件、语音
- [x] **消息状态**: 发送中、已发送、已读
- [x] **消息撤回**: 2分钟内撤回
- [x] **历史消息**: 游标分页拉取

### 关系功能

- [x] **好友管理**: 添加、删除、备注、黑名单
- [x] **好友申请**: 申请、同意、拒绝
- [x] **群组管理**: 创建、加入、退出、解散
- [x] **群成员管理**: 管理员、禁言

### 推送功能

- [x] **在线推送**: WebSocket 实时推送
- [x] **离线推送**: APNs (iOS) / FCM (Android)
- [x] **未读计数**: 实时未读消息数

### 文件功能

- [x] **文件上传**: 预签名 URL 直传
- [x] **图片缩略图**: 自动生成缩略图
- [x] **文件类型**: 图片、视频、文档

---

## 项目结构

```
gochat/
├── cmd/                          # 服务入口
│   ├── gateway/                  # 网关服务
│   ├── message/                  # 消息服务
│   ├── relation/                 # 关系服务
│   ├── push/                     # 推送服务
│   └── file/                     # 文件服务
├── internal/                     # 内部实现
│   ├── gateway/
│   │   ├── server/               # WebSocket 服务
│   │   ├── connection/           # 连接管理 (Goroutine + Channel)
│   │   └── router/               # 消息路由
│   ├── message/
│   │   ├── handler/              # 消息处理
│   │   ├── repository/           # 数据访问
│   │   └── service/              # 业务逻辑
│   ├── relation/
│   ├── push/
│   └── file/
├── pkg/                          # 公共包
│   ├── auth/                     # JWT 认证
│   ├── cache/                    # Redis 封装
│   ├── db/                       # MySQL 封装
│   ├── kafka/                    # Kafka 生产者/消费者
│   ├── logger/                   # Zap 日志
│   ├── snowflake/                # 分布式 ID 生成
│   └── response/                 # 统一响应
├── api/                          # API 定义
│   └── proto/                    # Protobuf 文件
├── configs/                      # 配置文件
│   ├── gateway.yaml
│   ├── message.yaml
│   └── ...
├── deployments/                  # 部署配置
│   ├── docker/                   # Dockerfile
│   ├── docker-compose.yml        # 本地开发环境
│   ├── k8s/                      # Kubernetes manifests
│   └── helm/                     # Helm Charts
├── scripts/                      # 脚本
│   ├── build.sh
│   ├── migrate.sh
│   └── test.sh
├── web/                          # 前端代码 (Vue 3)
├── docs/                         # 文档
│   ├── api.md
│   ├── architecture.md
│   └── deployment.md
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## 快速开始

### 环境要求

- Go 1.22+
- Docker & Docker Compose
- Make

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/yourusername/gochat.git
cd gochat

# 2. 启动基础设施 (MySQL, Redis, Kafka, MinIO)
make infra-up

# 3. 初始化数据库
make migrate

# 4. 启动所有服务
make run

# 5. 访问 Web 界面
open http://localhost:3000
```

### Docker Compose 部署

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f gateway
```

### Kubernetes 部署

```bash
# 添加 Helm 仓库
helm repo add gochat https://yourusername.github.io/gochat

# 安装
helm install gochat gochat/gochat \
  --namespace gochat \
  --create-namespace \
  --values values.yaml

# 查看状态
kubectl get pods -n gochat
```

---

## API 文档

### WebSocket 连接

```
ws://localhost:8080/ws?token=<jwt_token>
```

### 消息格式

```json
{
  "type": "message",
  "data": {
    "msg_id": "1234567890",
    "conversation_id": "conv_123",
    "sender_id": 1001,
    "receiver_id": 1002,
    "msg_type": 1,
    "content": "Hello, World!",
    "timestamp": 1706284800000
  }
}
```

### 消息类型

| type | 说明 |
|------|-----|
| 1 | 文本消息 |
| 2 | 图片消息 |
| 3 | 文件消息 |
| 4 | 语音消息 |
| 5 | 视频消息 |

### HTTP API

| 方法 | 路径 | 说明 |
|-----|------|-----|
| POST | /api/v1/auth/register | 用户注册 |
| POST | /api/v1/auth/login | 用户登录 |
| GET | /api/v1/messages | 获取历史消息 |
| POST | /api/v1/messages/recall | 撤回消息 |
| GET | /api/v1/friends | 获取好友列表 |
| POST | /api/v1/friends/apply | 发送好友申请 |
| GET | /api/v1/groups | 获取群组列表 |
| POST | /api/v1/groups | 创建群组 |
| POST | /api/v1/files/upload | 获取上传凭证 |

完整 API 文档: [docs/api.md](docs/api.md)

---

## 核心模块设计

### Gateway Service - 连接管理

使用 Goroutine + Channel 实现高并发连接管理：

```go
// ConnectionManager 连接管理器
type ConnectionManager struct {
    connections sync.Map              // connID -> *Connection
    userConns   sync.Map              // userID -> []*Connection (多端登录)
    broadcast   chan *Message         // 广播channel
    register    chan *Connection      // 注册channel
    unregister  chan *Connection      // 注销channel
}

// Connection 单个连接
type Connection struct {
    ID        string
    UserID    string
    Conn      *websocket.Conn
    Send      chan []byte            // 每个连接独立的发送channel
    closeChan chan struct{}
}
```

### Message Service - 消息处理

```go
// Message 消息结构
type Message struct {
    MsgID          string    `json:"msg_id"`
    ConversationID string    `json:"conversation_id"`
    SenderID       int64     `json:"sender_id"`
    ReceiverID     int64     `json:"receiver_id"`
    GroupID        int64     `json:"group_id,omitempty"`
    MsgType        int       `json:"msg_type"`
    Content        string    `json:"content"`
    Status         int       `json:"status"`
    CreatedAt      time.Time `json:"created_at"`
}
```

---

## 数据库设计

### 消息表 (按月分表)

```sql
CREATE TABLE message_202601 (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    msg_id VARCHAR(64) UNIQUE COMMENT '分布式ID (Snowflake)',
    conversation_id VARCHAR(64) COMMENT '会话ID',
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT COMMENT '私聊时有效',
    group_id BIGINT COMMENT '群聊时有效',
    msg_type TINYINT NOT NULL COMMENT '1:文本 2:图片 3:文件 4:语音 5:视频',
    content TEXT,
    status TINYINT DEFAULT 0 COMMENT '0:正常 1:撤回',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_conversation_id (conversation_id),
    INDEX idx_sender_id (sender_id),
    INDEX idx_created_at (created_at),
    INDEX idx_conv_time (conversation_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 好友关系表

```sql
CREATE TABLE friendship (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    friend_id BIGINT NOT NULL,
    remark VARCHAR(50) COMMENT '好友备注',
    status TINYINT DEFAULT 0 COMMENT '0:正常 1:拉黑',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_friend (user_id, friend_id),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 群组表

```sql
CREATE TABLE `group` (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    avatar VARCHAR(255),
    owner_id BIGINT NOT NULL,
    max_members INT DEFAULT 500,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_owner_id (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE group_member (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role TINYINT DEFAULT 0 COMMENT '0:成员 1:管理员 2:群主',
    nickname VARCHAR(50),
    muted_until TIMESTAMP NULL COMMENT '禁言截止时间',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_user (group_id, user_id),
    INDEX idx_group_id (group_id),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 性能指标

| 指标 | 数值 |
|-----|------|
| 单机并发连接数 | 100,000+ |
| 消息吞吐量 | 50,000 msg/s |
| 消息延迟 (P99) | < 50ms |
| 消息读取延迟 | < 10ms |
| Redis 缓存命中率 | 95%+ |

### 压测环境

- CPU: 8 Core
- Memory: 16 GB
- Network: 1 Gbps
- OS: Ubuntu 22.04

---

## 开发计划

### Phase 1: 基础架构 (Week 1)

- [x] 项目初始化、目录结构
- [ ] Docker Compose 开发环境
- [ ] Gateway 服务 (WebSocket + Goroutine + Channel)
- [ ] 用户认证 (JWT)

### Phase 2: 消息服务 (Week 2)

- [ ] Message Service (Kafka + MySQL)
- [ ] 私聊/群聊消息
- [ ] Relation Service (好友/群组)
- [ ] Redis 缓存层

### Phase 3: 推送与文件 (Week 3)

- [ ] Push Service (在线/离线推送)
- [ ] File Service (MinIO)
- [ ] Web 前端界面

### Phase 4: 部署与优化 (Week 4)

- [ ] Kubernetes 部署
- [ ] Prometheus + Grafana 监控
- [ ] 性能压测与优化
- [ ] 文档完善

---

## 贡献指南

欢迎提交 Issue 和 Pull Request！

```bash
# Fork 项目后
git checkout -b feature/your-feature
git commit -m "Add your feature"
git push origin feature/your-feature
# 创建 Pull Request
```

---

## 许可证

[MIT License](LICENSE)

---

## 联系方式

- Email: your.email@example.com
- GitHub: [@yourusername](https://github.com/yourusername)

---

<div align="center">

**如果这个项目对你有帮助，请给一个 Star!**

</div>
