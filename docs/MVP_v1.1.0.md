# GoChat v1.1.0 - MVP Release

> 当前版本为 **MVP (Minimum Viable Product)**，实现了即时通讯系统的核心功能。

## 📦 当前实现的功能 (v1.1.0)

### ✅ 已实现
- **用户系统** - 注册、登录、JWT 认证、Session 持久化
- **好友管理** - 添加好友、好友申请、删除好友、备注修改
- **拉黑系统** - 拉黑/取消拉黑、状态实时推送、双向状态展示
- **一对一聊天** - WebSocket 实时聊天、消息持久化、历史消息查询
- **实时通知** - 关系变更实时推送（基于 Kafka）
- **Web UI** - QQ 风格界面，响应式设计

### ❌ 尚未实现（计划中）
- 群聊功能
- 图片/文件消息
- 消息已读回执
- 消息撤回
- 语音/视频通话
- 离线推送 (APNs/FCM)
- 多端同步
- 服务注册发现 (Consul/Etcd)

---

## 🏗️ 当前架构

```
┌─────────────────────────────────────────────────────────┐
│                    GoChat v1.1.0 (MVP)                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │  Gateway │  │ Relation │  │ Message  │  │  User  ││
│  │  :8080   │  │  :8082   │  │  :8081   │  │ :8085  ││
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬────┘│
│       │             │             │            │      │
│       └─────────────┴─────────────┴────────────┘      │
│                         │                             │
│                   ┌─────▼─────┐                     │
│                   │   Kafka   │                     │
│                   │ :9092     │                     │
│                   └─────┬─────┘                     │
│                         │                             │
│                   ┌─────▼─────┐                     │
│                   │   MySQL   │                     │
│                   │ :3306     │                     │
│                   └───────────┘                     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 🚀 快速启动

### 1. 启动基础设施
```bash
docker-compose up -d mysql kafka
```

### 2. 创建数据库
```bash
docker exec -it gochat-mysql mysql -uroot
CREATE DATABASE gochat DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
EXIT;
```

### 3. 创建 Kafka Topics
```bash
docker exec gochat-kafka kafka-topics --bootstrap-server localhost:9092 \
  --create --topic message-topic --partitions 3 --replication-factor 1

docker exec gochat-kafka kafka-topics --bootstrap-server localhost:9092 \
  --create --topic relation-topic --partitions 3 --replication-factor 1
```

### 4. 编译并启动服务
```bash
# 编译
go build -o bin/gateway.exe ./cmd/gateway
go build -o bin/message.exe ./cmd/message
go build -o bin/relation.exe ./cmd/relation
go build -o bin/user.exe ./cmd/user

# 启动（分别打开 4 个终端）
bin\gateway.exe
bin\message.exe
bin\relation.exe
bin\user.exe
```

### 5. 访问前端
```
http://localhost:8080
```

---

## 📖 API 文档

### User Service (:8085)
```
POST   /api/v1/register           用户注册
POST   /api/v1/login              用户登录
GET    /api/v1/users/search       搜索用户
```

### Relation Service (:8082)
```
POST   /api/v1/friend/apply                     发送好友申请
POST   /api/v1/friend/apply/handle               处理好友申请
GET    /api/v1/friend/apply/list                获取申请列表
GET    /api/v1/friends                          获取好友列表
DELETE /api/v1/friends/{id}                     删除好友
POST   /api/v1/friends/{id}/block               拉黑好友
POST   /api/v1/friends/{id}/unblock             取消拉黑
PUT    /api/v1/friends/remark                    修改备注
```

### Message Service (:8081)
```
GET    /api/v1/messages              获取消息历史
GET    /api/v1/conversations         获取会话列表
```

### Gateway Service (:8080)
```
WS     /ws                         WebSocket 连接
GET    /health                     健康检查
```

---

## 🔧 配置文件

| 配置文件 | 说明 |
|---------|------|
| `configs/gateway.yaml` | Gateway 服务配置 |
| `configs/message.yaml` | Message 服务配置 |
| `configs/relation.yaml` | Relation 服务配置 |
| `configs/user.yaml` | User 服务配置 |

---

## 📊 数据库表

### user 表
```sql
CREATE TABLE user (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  password VARCHAR(255) NOT NULL,
  nickname VARCHAR(50),
  avatar VARCHAR(255),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### friendship 表
```sql
CREATE TABLE friendship (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  friend_id BIGINT NOT NULL,
  remark VARCHAR(50),
  status INT DEFAULT 0,  -- 0=正常, 1=拉黑
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_friend (user_id, friend_id)
);
```

### message_YYYYMM 表（按月分表）
```sql
CREATE TABLE message_202601 (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  msg_id VARCHAR(64) UNIQUE NOT NULL,
  conversation_id VARCHAR(64) NOT NULL,
  sender_id BIGINT NOT NULL,
  receiver_id BIGINT,
  msg_type INT DEFAULT 1,
  content TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  KEY idx_conversation (conversation_id),
  KEY idx_created (created_at)
);
```

---

## 🎯 测试流程

### 1. 注册/登录
1. 打开 http://localhost:8080
2. 注册两个用户（User A 和 User B）
3. 分别登录

### 2. 添加好友
1. User A 搜索 User B
2. 点击"添加"，发送好友申请
3. User B 登录，在"申请"标签中接受申请

### 3. 拉黑测试
1. User A 和 User B 互相发送消息
2. User A 点击聊天头部的"⋮"菜单，选择"拉黑用户"
3. User B 立即看到红色警告条：`对方已将你拉黑，消息无法送达`
4. 双方发送的消息仅本地显示
5. User A 取消拉黑，User B 立即看到提示：`对方已取消对你的拉黑`

---

## 📝 版本历史

- **v1.1.0** (2026-02-01) - 拉黑系统 + 实时推送
- **v1.0.0** (2026-01-30) - MVP 基础版本

详细变更记录请参考 [CHANGELOG.md](CHANGELOG.md)

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

## 📄 许可证

MIT License
