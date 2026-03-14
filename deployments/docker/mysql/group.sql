SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS `group` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `avatar` VARCHAR(255) DEFAULT '' COMMENT 'avatur_url',
  `description` VARCHAR(500) DEFAULT '',
  `owner_id` BIGINT NOT NULL COMMENT 'foreign key to user.id',
  `max_members` INT DEFAULT 500,
  `member_count` INT DEFAULT 1,
  `announcement` VARCHAR(1000) DEFAULT '',
  `status` TINYINT DEFAULT 1 COMMENT 'status: 0closed 1normal',
  `meta_version` BIGINT NOT NULL DEFAULT 0 COMMENT 'group name and announcement version',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_owner_id` (`owner_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `group_member` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `role` TINYINT DEFAULT 0 COMMENT 'role: 0member 1adminstrator 2owner',
  `nickname` VARCHAR(50) DEFAULT '' COMMENT 'nickname in group',
  `info_version` BIGINT NOT NULL DEFAULT 0 COMMENT 'group member nickname version',
  `muted_until` TIMESTAMP NULL DEFAULT NULL COMMENT 'only before this time can send message, controlled by admin or owner',
  `joined_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_group_user` (`group_id`, `user_id`),
  KEY `idx_user_group_role` (`user_id`, `group_id`, `role`),
  KEY `idx_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `group_request` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `message` VARCHAR(255) DEFAULT '' COMMENT 'application message',
  `status` TINYINT DEFAULT 0 COMMENT 'status: 0pending 1approved 2rejected',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;