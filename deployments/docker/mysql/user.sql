SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS `user` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(50) NOT NULL,
  `password` VARCHAR(255) NOT NULL COMMENT 'password_hash (bcrypt)',
  `nickname` VARCHAR(50) NOT NULL,
  `avatar` VARCHAR(255) DEFAULT '' COMMENT 'avatar_URL',
  `phone` VARCHAR(20) DEFAULT '',
  `email` VARCHAR(100) DEFAULT '',
  `gender` TINYINT DEFAULT 0,
  `status` TINYINT DEFAULT 1 COMMENT 'status: 0blacklisted 1normal',
  `info_version` BIGINT NOT NULL DEFAULT 0 COMMENT 'user info version',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_phone` (`phone`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user_device` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `device_id` VARCHAR(100) NOT NULL,
  `platform` VARCHAR(20) NOT NULL,
  `push_token` VARCHAR(255) DEFAULT '',
  `last_active_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_device` (`user_id`, `device_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;