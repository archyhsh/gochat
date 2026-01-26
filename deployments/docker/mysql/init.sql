-- GoChat 数据库初始化脚本
-- 创建时间: 2024

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- 用户表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(50) NOT NULL COMMENT '用户名',
  `password` VARCHAR(255) NOT NULL COMMENT '密码 (bcrypt)',
  `nickname` VARCHAR(50) NOT NULL COMMENT '昵称',
  `avatar` VARCHAR(255) DEFAULT '' COMMENT '头像URL',
  `phone` VARCHAR(20) DEFAULT '' COMMENT '手机号',
  `email` VARCHAR(100) DEFAULT '' COMMENT '邮箱',
  `gender` TINYINT DEFAULT 0 COMMENT '性别: 0未知 1男 2女',
  `status` TINYINT DEFAULT 1 COMMENT '状态: 0禁用 1正常',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_phone` (`phone`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- ----------------------------
-- 好友关系表 (双向存储)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `friendship` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `friend_id` BIGINT NOT NULL COMMENT '好友ID',
  `remark` VARCHAR(50) DEFAULT '' COMMENT '好友备注',
  `status` TINYINT DEFAULT 0 COMMENT '状态: 0正常 1拉黑',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_friend_id` (`friend_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友关系表';

-- ----------------------------
-- 好友申请表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `friend_apply` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `from_user_id` BIGINT NOT NULL COMMENT '申请人ID',
  `to_user_id` BIGINT NOT NULL COMMENT '被申请人ID',
  `message` VARCHAR(255) DEFAULT '' COMMENT '申请消息',
  `status` TINYINT DEFAULT 0 COMMENT '状态: 0待处理 1已同意 2已拒绝',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_from_user` (`from_user_id`),
  KEY `idx_to_user` (`to_user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友申请表';

-- ----------------------------
-- 群组表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `group` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '群名称',
  `avatar` VARCHAR(255) DEFAULT '' COMMENT '群头像',
  `description` VARCHAR(500) DEFAULT '' COMMENT '群描述',
  `owner_id` BIGINT NOT NULL COMMENT '群主ID',
  `max_members` INT DEFAULT 500 COMMENT '最大成员数',
  `member_count` INT DEFAULT 1 COMMENT '当前成员数',
  `status` TINYINT DEFAULT 1 COMMENT '状态: 0解散 1正常',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_owner_id` (`owner_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群组表';

-- ----------------------------
-- 群成员表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `group_member` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT NOT NULL COMMENT '群ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `role` TINYINT DEFAULT 0 COMMENT '角色: 0成员 1管理员 2群主',
  `nickname` VARCHAR(50) DEFAULT '' COMMENT '群内昵称',
  `muted_until` TIMESTAMP NULL COMMENT '禁言截止时间',
  `joined_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_group_user` (`group_id`, `user_id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群成员表';

-- ----------------------------
-- 会话表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `conversation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `conversation_id` VARCHAR(64) NOT NULL COMMENT '会话ID',
  `type` TINYINT NOT NULL COMMENT '类型: 1私聊 2群聊',
  `target_id` BIGINT NOT NULL COMMENT '目标ID (user_id或group_id)',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_conversation_id` (`conversation_id`),
  KEY `idx_type_target` (`type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话表';

-- ----------------------------
-- 用户会话表 (存储用户的会话列表)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user_conversation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `conversation_id` VARCHAR(64) NOT NULL COMMENT '会话ID',
  `unread_count` INT DEFAULT 0 COMMENT '未读消息数',
  `last_msg_id` VARCHAR(64) DEFAULT '' COMMENT '最后一条消息ID',
  `last_msg_time` TIMESTAMP NULL COMMENT '最后消息时间',
  `is_top` TINYINT DEFAULT 0 COMMENT '是否置顶',
  `is_muted` TINYINT DEFAULT 0 COMMENT '是否免打扰',
  `is_deleted` TINYINT DEFAULT 0 COMMENT '是否删除',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_conv` (`user_id`, `conversation_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_last_msg_time` (`last_msg_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户会话表';

-- ----------------------------
-- 消息表模板 (按月分表)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `message_template` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `msg_id` VARCHAR(64) NOT NULL COMMENT '消息ID (Snowflake)',
  `conversation_id` VARCHAR(64) NOT NULL COMMENT '会话ID',
  `sender_id` BIGINT NOT NULL COMMENT '发送者ID',
  `receiver_id` BIGINT DEFAULT 0 COMMENT '接收者ID (私聊)',
  `group_id` BIGINT DEFAULT 0 COMMENT '群ID (群聊)',
  `msg_type` TINYINT NOT NULL COMMENT '消息类型: 1文本 2图片 3文件 4语音 5视频 6系统',
  `content` TEXT COMMENT '消息内容 (JSON格式)',
  `status` TINYINT DEFAULT 0 COMMENT '状态: 0正常 1撤回',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_msg_id` (`msg_id`),
  KEY `idx_conversation_id` (`conversation_id`),
  KEY `idx_sender_id` (`sender_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_conv_time` (`conversation_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息表模板';

-- 创建当月消息表
CREATE TABLE IF NOT EXISTS `message_202601` LIKE `message_template`;

-- ----------------------------
-- 消息已读状态表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `message_read` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `msg_id` VARCHAR(64) NOT NULL COMMENT '消息ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `read_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '已读时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_msg_user` (`msg_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息已读状态表';

-- ----------------------------
-- 文件表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `file` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `file_id` VARCHAR(64) NOT NULL COMMENT '文件ID',
  `user_id` BIGINT NOT NULL COMMENT '上传者ID',
  `filename` VARCHAR(255) NOT NULL COMMENT '原始文件名',
  `file_path` VARCHAR(500) NOT NULL COMMENT '存储路径',
  `file_size` BIGINT NOT NULL COMMENT '文件大小 (bytes)',
  `file_type` VARCHAR(100) NOT NULL COMMENT 'MIME类型',
  `thumbnail_path` VARCHAR(500) DEFAULT '' COMMENT '缩略图路径',
  `md5` VARCHAR(32) DEFAULT '' COMMENT '文件MD5',
  `status` TINYINT DEFAULT 1 COMMENT '状态: 0删除 1正常',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_id` (`file_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_md5` (`md5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件表';

-- ----------------------------
-- 用户设备表 (推送)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user_device` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `device_id` VARCHAR(100) NOT NULL COMMENT '设备ID',
  `platform` VARCHAR(20) NOT NULL COMMENT '平台: ios/android/web',
  `push_token` VARCHAR(255) DEFAULT '' COMMENT '推送Token',
  `last_active_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '最后活跃时间',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_device` (`user_id`, `device_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户设备表';

SET FOREIGN_KEY_CHECKS = 1;

-- 插入测试用户
INSERT INTO `user` (`id`, `username`, `password`, `nickname`) VALUES
(1, 'admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '管理员'),
(2, 'test1', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '测试用户1'),
(3, 'test2', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '测试用户2');
-- 密码均为: password123
