# GoChat TODO

## 已完成模块

### Gateway 服务 ✅
- [x] WebSocket 连接管理 (connection/manager.go, connection.go)
- [x] 消息处理器 (handler/message.go)
- [x] 服务启动入口 (cmd/gateway/main.go)
- [x] Kafka Producer 集成
- [x] 测试 Token 生成接口
- [x] 前端演示页面 (web/static/index.html)

### Message 服务 ✅
- [x] 消息数据模型 (model/message.go)
- [x] 消息服务层 (service/message.go)
- [x] Kafka Consumer (consumer/consumer.go)
- [x] 消息持久化到 MySQL
- [x] 服务启动入口 (cmd/message/main.go)

### 基础设施 ✅
- [x] MySQL 数据库表设计 (deployments/docker/mysql/init.sql)
- [x] Docker Compose 配置
- [x] 配置文件 (configs/*.yaml)

---

## 待开发模块

### 用户服务 (User Service)
- [ ] 用户注册接口
- [ ] 用户登录接口 (返回 JWT)
- [ ] 用户信息查询/修改
- [ ] 密码修改

### 关系服务 (Relation Service)
- [ ] 好友申请
- [ ] 好友列表
- [ ] 好友删除/拉黑
- [ ] 好友备注

### 群聊功能
- [ ] 创建群组
- [ ] 群成员管理
- [ ] 群消息广播
- [ ] 群公告

### 推送服务 (Push Service)
- [ ] 离线消息存储
- [ ] APNs/FCM 推送集成

---

## 待完善功能

### 入口限流 (Rate Limiting)
- [ ] 用户发消息频率限制
- [ ] 服务端过载保护
- [ ] 返回明确错误码

### 消息补偿拉取
- [x] 消息持久化到 DB/MQ ✅
- [ ] 客户端定期同步，补漏缺失消息
- [ ] 消息 ACK 确认机制

### 连接健康管理
- [ ] 缓冲区频繁满时主动断开慢客户端
- [ ] 连接质量监控与告警

### 消息处理增强
- [x] handleChat: 发送到 Kafka ✅
- [x] handleChat: 持久化到数据库 ✅
- [ ] handleChat: 群聊消息广播
- [ ] handleAck: 更新消息状态
- [ ] handleRead: 更新已读状态
- [ ] handleRead: 通知发送者
- [ ] handleTyping: 通知对方正在输入

### 安全增强
- [ ] WebSocket CheckOrigin 白名单
- [ ] 生产环境移除测试 Token 接口

---

## 项目结构

```
gochat/
├── cmd/
│   ├── gateway/main.go      # Gateway 服务入口
│   └── message/main.go      # Message 服务入口
├── configs/
│   ├── gateway.yaml
│   └── message.yaml
├── deployments/
│   ├── docker-compose.yml
│   └── docker/
├── internal/
│   ├── gateway/
│   │   ├── connection/      # WebSocket 连接管理
│   │   ├── handler/         # 消息处理
│   │   └── server/          # HTTP 服务
│   └── message/
│       ├── consumer/        # Kafka 消费者
│       ├── model/           # 数据模型
│       └── service/         # 业务逻辑
├── pkg/                     # 公共库
├── web/static/              # 前端演示
└── bin/                     # 编译产物
```
