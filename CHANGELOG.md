# Changelog

All notable changes to GoChat will be documented in this file.

## [1.1.0] - 2026-02-01

### Added
- **拉黑系统**
  - 拉黑好友功能 (`POST /api/v1/friends/{id}/block`)
  - 取消拉黑功能 (`POST /api/v1/friends/{id}/unblock`)
  - 双向拉黑状态显示（已拉黑/被拉黑/互相拉黑）
  - 拉黑后消息仅本地可见

- **实时关系变更推送**
  - 基于 Kafka 的事件驱动架构
  - 拉黑/取消拉黑实时通知对方
  - 无需刷新页面即可看到状态变化
  - 新增 Kafka topic: `relation-topic`

- **Session 持久化**
  - localStorage 保存登录状态
  - 刷新页面自动登录
  - Token 过期自动跳转登录页

- **消息历史查询**
  - 支持加载历史消息（分页）
  - 跨月份表查询（最近3个月）
  - 会话列表展示

- **完整前端 UI**
  - QQ 风格界面设计
  - 响应式布局
  - 好友列表、聊天列表、好友申请列表
  - 用户搜索功能
  - 聊天界面（消息气泡、输入框、加载更多）

### Changed
- 重构消息路由逻辑，支持拉黑状态检查
- 优化前端代码结构，提升可维护性

### Fixed
- 修复拉黑后消息仍发送到对方的问题
- 修复取消拉黑后对方无法实时收到通知的问题
- 修复 Session 丢失导致频繁登录的问题

### Technical Details
- **Kafka Topics:**
  - `message-topic` - 聊天消息
  - `relation-topic` - 关系变更事件 (新增)

- **Database Schema:**
  - `friendship` 表新增 `status` 字段（0=正常, 1=拉黑）
  - 消息表按月分表: `message_YYYYMM`

- **WebSocket Events:**
  - `chat` - 聊天消息
  - `relation_changed` - 关系变更推送 (新增)

---

## [1.0.0] - 2026-01-30

### Added
- **用户系统**
  - 用户注册 (`POST /api/v1/register`)
  - 用户登录 (`POST /api/v1/login`)
  - JWT 认证
  - 用户搜索 (`GET /api/v1/users/search`)

- **好友系统**
  - 发送好友申请 (`POST /api/v1/friend/apply`)
  - 处理好友申请 (`POST /api/v1/friend/apply/handle`)
  - 获取好友列表 (`GET /api/v1/friends`)
  - 获取申请列表 (`GET /api/v1/friend/apply/list`)
  - 删除好友 (`DELETE /api/v1/friends/{id}`)
  - 修改备注 (`PUT /api/v1/friends/remark`)

- **即时通讯**
  - WebSocket 实时聊天
  - 消息持久化（Kafka + MySQL）
  - 消息查询 (`GET /api/v1/messages`)
  - 会话列表 (`GET /api/v1/conversations`)

- **微服务架构**
  - Gateway Service (`:8080`) - WebSocket 网关
  - Message Service (`:8081`) - 消息服务
  - Relation Service (`:8082`) - 关系服务
  - User Service (`:8085`) - 用户服务

### Infrastructure
- MySQL 8.0 - 数据存储
- Kafka 7.5.0 - 消息队列
- Docker & Docker Compose - 容器化部署

---

## [Unreleased]

### Planned
- 群聊功能
- 图片/文件消息
- 消息已读回执
- 消息撤回
- 语音/视频通话
- @提及功能

---

## 版本号规则

遵循 [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR.MINOR.PATCH**
- MAJOR: 不兼容的 API 变更
- MINOR: 向后兼容的功能新增
- PATCH: 向后兼容的问题修复

示例：
- `1.0.0` → `1.1.0` - 新增拉黑功能
- `1.1.0` → `1.2.0` - 新增群聊功能
- `1.2.0` → `2.0.0` - 架构重构，不兼容变更
