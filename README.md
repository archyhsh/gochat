# GoChat v2.0 (Cluster Edition) — 分布式工业级即时通讯系统

GoChat v2.0 是一款基于 **go-zero** 微服务框架、采用云原生架构设计的分布式 IM 系统。本项目从 1.0 的单点原型全面进化为具备 **高可用、高性能、全链路追踪** 能力的生产级后端。

## 🚀 核心架构演进 (v2.0 亮点)

### 1. 分布式集群架构 (Distributed Cluster)
*   **网关横向扩展**：支持多 Gateway 节点部署，通过 Redis 实现跨节点的用户路由寻址与长连接管理。
*   **动态服务发现**：全面接入 **Etcd** 注册中心，实现微服务间的自动发现与负载均衡，彻底废弃硬编码地址。
*   **智能消息分发**：利用 Kafka 分区 Key 策略，确保同一会话的消息在分布式消费时具备局部顺序性。

### 2. 极致可靠的消息体系 (Reliability)
*   **可靠发送 (SafeSend)**：内置应用层指数退避重试逻辑，解决 Kafka 瞬时故障导致的消息丢失。
*   **死信队列 (DLQ)**：集成 **Redis DLQ** 体系与自动化补偿脚本 `dlq_recovery.go`，确保极端故障下的数据最终一致性。
*   **分布式同步**：利用 Kafka 广播模式解决了多节点环境下 L1 本地内存缓存的一致性挑战。

### 3. 高并发大群聊优化 (Performance)
*   **推代拉模式**：通过“写即预热”策略，使大群聊的会话列表请求 **100% 命中 Redis**，极大保护了 MySQL。
*   **寻址批量化**：引入 Redis `MGET` (BatchFind) 优化，将大群消息分发的 I/O 损耗降低了 90% 以上。
*   **静默化推送**：对成员进出等低频信号执行静默持久化，彻底杜绝 500 人大群中的流量风暴。

### 4. 全方位安全与观测 (Security & Observability)
*   **全栈权限治理**：基于 gRPC Metadata 实现了统一的身份校验，修复了消息窃取、垂直越权等高危漏洞，并实现了敏感数据脱敏。
*   **全景观测栈 (OTLP)**：全线接入 **OTLP gRPC** 协议。
    *   **Jaeger**: 实时呈现跨网关、Kafka、RPC 的完整调用拓扑。
    *   **Prometheus & Grafana**: 提供全业务维度的性能指标监控。

---

## 🛠 快速启动 (一键集群化部署)

确保您的环境已安装 Docker (v20.10+) 和 Docker Compose。

```bash
# 1. 进入部署目录
cd deployments

# 2. 一键拉起全量集群（含基础设施、微服务及监控栈）
docker-compose -f docker-compose.cluster.yml up --build -d
```

### 访问入口
| 组件 | 访问地址 | 备注 |
| :--- | :--- | :--- |
| **前端入口 1** | [http://localhost:8881](http://localhost:8881) | 网关节点 A (已实装 CORS) |
| **前端入口 2** | [http://localhost:8882](http://localhost:8882) | 网关节点 B (开发热挂载) |
| **Jaeger UI** | [http://localhost:16686](http://localhost:16686) | 查看链路追踪与延迟 |
| **Grafana** | [http://localhost:3001](http://localhost:3001) | 可视化监控 (admin/admin) |
| **Prometheus** | [http://localhost:9090](http://localhost:9090) | 指标原始数据查询 |

---

## 🏗 微服务矩阵
- **Gateway**: WebSocket 长连接管理、CORS 支持、审计日志、多级路由分发。
- **User RPC**: 用户生命周期、隐私保护、分布式版本控制。
- **Message RPC**: 高性能消息持久化、多级缓存预热、分表存储、原子计数器。
- **Relation/Group RPC**: 好友关系链与群组业务逻辑。

---
*本项目仅供技术交流与学习使用。*
