# GoChat v1.1.0 (MVP) - 快速启动指南

## 1. 启动基础环境
确保已安装 Docker 和 Docker Compose。

```bash
# 启动 MySQL 和 Kafka
docker-compose up -d mysql kafka
sql 重构后 请使用 docker-compose down -v
```

## 2. 初始化数据库
连接到 MySQL 并创建名为 `gochat` 的数据库。

```bash
docker exec -it gochat-mysql mysql -uroot -pgochat123 -e "CREATE DATABASE IF NOT EXISTS gochat;"
```

## 3. 初始化 Kafka Topics
创建消息分发所需的 Topic。

```bash
# 创建消息 Topic
docker exec gochat-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic message-topic --partitions 3 --replication-factor 1

# 创建关系变更 Topic
docker exec gochat-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic relation-topic --partitions 3 --replication-factor 1
```

## 4. 编译服务
```bash
go build -o bin/gateway.exe ./cmd/gateway
go build -o bin/message.exe ./cmd/message
go build -o bin/relation.exe ./cmd/relation
go build -o bin/user.exe ./cmd/user
```

## 5. 启动服务
建议在 4 个独立的终端窗口中分别运行以下命令：

```bash
# 启动用户服务 (端口 8085)
bin\user.exe

# 启动关系服务 (端口 8082)
bin\relation.exe

# 启动消息服务 (端口 8081)
bin\message.exe

# 启动网关服务 (端口 8080)
bin\gateway.exe
```

## 6. 访问系统
打开浏览器访问：
[http://localhost:8080](http://localhost:8080)

---

### 系统模块说明
- **Gateway (8080)**: 负责 WebSocket 连接管理与消息实时推送。
- **User (8085)**: 负责用户注册、登录及鉴权。
- **Relation (8082)**: 负责好友管理、群组管理及关系变更通知。
- **Message (8081)**: 负责聊天记录持久化与历史消息查询。
