SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 创建核心业务数据库
CREATE DATABASE IF NOT EXISTS `gochat_user` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `gochat_group` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `gochat_relation` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `gochat_message` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 显式授予 gochat 用户对所有业务库的权限
-- 注意：docker-compose 中已经创建了 gochat 用户，这里只需补全授权
GRANT ALL PRIVILEGES ON `gochat_user`.* TO 'gochat'@'%';
GRANT ALL PRIVILEGES ON `gochat_group`.* TO 'gochat'@'%';
GRANT ALL PRIVILEGES ON `gochat_relation`.* TO 'gochat'@'%';
GRANT ALL PRIVILEGES ON `gochat_message`.* TO 'gochat'@'%';
FLUSH PRIVILEGES;
