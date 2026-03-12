SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS `conversation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `conversation_id` VARCHAR(64) NOT NULL,
  `type` TINYINT NOT NULL COMMENT 'type: 1private 2group',
  `target_id` BIGINT NOT NULL COMMENT 'user_id or group_id',
  `last_msg_id` VARCHAR(64) NOT NULL DEFAULT '',
  `last_msg_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_msg_content` TEXT NOT NULL,
  `last_msg_type` TINYINT NOT NULL DEFAULT 0,
  `last_sender_id` BIGINT NOT NULL DEFAULT 0,
  `latest_seq` BIGINT NOT NULL DEFAULT 0 COMMENT 'latest msg sequence',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_conversation_id` (`conversation_id`),
  KEY `idx_type_target` (`type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS `user_conversation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `conversation_id` VARCHAR(64) NOT NULL,
  `peer_id` BIGINT NOT NULL DEFAULT 0 COMMENT 'Receiver or Group ID',
  `unread_count` INT DEFAULT 0,
  `last_msg_id` VARCHAR(64) NOT NULL DEFAULT '',
  `last_msg_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_msg_content` TEXT NOT NULL,
  `last_msg_type` TINYINT NOT NULL DEFAULT 0,
  `last_sender_id` BIGINT NOT NULL DEFAULT 0,
  `read_sequence` BIGINT NOT NULL DEFAULT 0 COMMENT 'last read msg sequence',
  `is_top` TINYINT NOT NULL DEFAULT 0,
  `is_muted` TINYINT NOT NULL DEFAULT 0,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `version` BIGINT NOT NULL DEFAULT 0 COMMENT 'delete/top/muted version(for multiple devices)',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_conv` (`user_id`, `conversation_id`),
  KEY `idx_user_active_list` (`user_id`, `is_deleted`, `last_msg_time` DESC),
  KEY `idx_last_msg_time` (`last_msg_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS `message_template` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `msg_id` VARCHAR(64) NOT NULL,
  `conversation_id` VARCHAR(64) NOT NULL,
  `sender_id` BIGINT NOT NULL,
  `receiver_id` BIGINT DEFAULT 0 COMMENT 'only for private chat',
  `group_id` BIGINT DEFAULT 0 COMMENT 'only for group chat',
  `sequence_id` BIGINT NOT NULL DEFAULT 0 COMMENT 'message sequence in conversation',
  `msg_type` TINYINT NOT NULL COMMENT 'type: 1text 2image 3file 4audio 5video 6system',
  `content` TEXT NOT NULL DEFAULT '' COMMENT 'JSON format content',
  `status` TINYINT DEFAULT 0 COMMENT 'status: 0normal 1withdrawn',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_msg_id` (`msg_id`),
  KEY `idx_conv_created` (`conversation_id`, `created_at` DESC),
  KEY `idx_sender_id` (`sender_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_conv_seq` (`group_id`, `sequence_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `message_read` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `msg_id` VARCHAR(64) NOT NULL,
  `user_id` BIGINT NOT NULL,
  `read_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_msg_user` (`msg_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;