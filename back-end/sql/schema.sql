-- Social Dorian MySQL schema
-- Database: doriansocial
--
-- Apply with:
--   mysql -u root -p < sql/schema.sql
-- or:
--   mysql -u root -p -e "SOURCE /path/to/back-end/sql/schema.sql"

CREATE DATABASE IF NOT EXISTS `doriansocial`
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE `doriansocial`;

-- Registered proxies (Proxy management)
CREATE TABLE IF NOT EXISTS `proxies` (
  `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name`       VARCHAR(120) NOT NULL,
  `protocol`   ENUM('http', 'https', 'socks5') NOT NULL DEFAULT 'http',
  `host`       VARCHAR(255) NOT NULL,
  `port`       INT UNSIGNED NOT NULL,
  `username`   VARCHAR(120) NOT NULL DEFAULT '',
  `password`   VARCHAR(255) NOT NULL DEFAULT '',
  `country`    VARCHAR(80)  NOT NULL DEFAULT '',
  `status`     ENUM('active', 'inactive', 'error') NOT NULL DEFAULT 'active',
  `notes`      TEXT         NOT NULL,
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_proxies_status` (`status`),
  KEY `idx_proxies_protocol` (`protocol`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Registered Gmail mailboxes (Gmail management)
CREATE TABLE IF NOT EXISTS `gmails` (
  `id`              INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `email`           VARCHAR(255) NOT NULL,
  `password`        VARCHAR(255) NOT NULL DEFAULT '',
  `app_password`    VARCHAR(255) NOT NULL,
  `recovery_email`  VARCHAR(255) NOT NULL DEFAULT '',
  `recovery_phone`  VARCHAR(60)  NOT NULL DEFAULT '',
  `twofa_secret`    VARCHAR(255) NOT NULL DEFAULT '',
  `backup_codes`    TEXT         NOT NULL,
  `pin_code`        VARCHAR(16) NOT NULL DEFAULT '',
  `label`           VARCHAR(120) NOT NULL DEFAULT '',
  `status`          ENUM('active', 'inactive', 'error') NOT NULL DEFAULT 'active',
  `notes`           TEXT         NOT NULL,
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gmails_email` (`email`),
  KEY `idx_gmails_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Admin users for social.dorian.center
CREATE TABLE IF NOT EXISTS `users` (
  `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `email`         VARCHAR(255) NOT NULL,
  `name`          VARCHAR(120) NOT NULL DEFAULT '',
  `password_hash` VARCHAR(255) NOT NULL,
  `role`          ENUM('admin', 'member') NOT NULL DEFAULT 'admin',
  `status`        ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_users_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `auth_sessions` (
  `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id`    INT UNSIGNED NOT NULL,
  `token_hash` CHAR(64) NOT NULL,
  `expires_at` DATETIME NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_auth_sessions_token` (`token_hash`),
  KEY `idx_auth_sessions_user` (`user_id`),
  KEY `idx_auth_sessions_expires` (`expires_at`),
  CONSTRAINT `fk_auth_sessions_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Connected social accounts
-- Account-to-proxy assignment is stored here: accounts.proxy_id -> proxies.id
-- One proxy can be assigned to many accounts (auto or manual mode).
-- Account email may match a registered Gmail mailbox in `gmails.email`.
CREATE TABLE IF NOT EXISTS `accounts` (
  `id`             INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `platform`       ENUM('twitter', 'instagram', 'linkedin', 'facebook', 'tiktok', 'youtube') NOT NULL,
  `first_name`     VARCHAR(120) NOT NULL,
  `last_name`      VARCHAR(120) NOT NULL,
  `password`       VARCHAR(255) NOT NULL,
  `email`          VARCHAR(255) NOT NULL,
  `email_password` VARCHAR(255) NOT NULL DEFAULT '',
  `birthday`       DATE         NULL,
  `gender`         ENUM('male', 'female', 'other', 'prefer_not_to_say') NOT NULL DEFAULT 'prefer_not_to_say',
  `proxy_mode`     ENUM('none', 'auto', 'manual') NOT NULL DEFAULT 'none',
  `proxy_id`       INT UNSIGNED NULL,
  `status`         ENUM('active', 'expired', 'error', 'revoked') NOT NULL DEFAULT 'active',
  `connected_by`   VARCHAR(120) NOT NULL DEFAULT '',
  `connected_at`   DATE         NOT NULL,
  `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_accounts_platform` (`platform`),
  KEY `idx_accounts_status` (`status`),
  KEY `idx_accounts_email` (`email`),
  KEY `idx_accounts_proxy_id` (`proxy_id`),
  CONSTRAINT `fk_accounts_proxy`
    FOREIGN KEY (`proxy_id`) REFERENCES `proxies` (`id`)
    ON DELETE SET NULL
    ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed proxies
INSERT INTO `proxies` (`id`, `name`, `protocol`, `host`, `port`, `username`, `password`, `country`, `status`, `notes`) VALUES
  (1, 'US East', 'http', 'us.proxy.dorian', 8080, 'dorian', 'secret', 'US', 'active', 'Primary residential pool'),
  (2, 'EU West', 'https', 'eu.proxy.dorian', 8443, 'dorian', 'secret', 'DE', 'active', 'EU compliance traffic'),
  (3, 'Local SOCKS', 'socks5', '127.0.0.1', 1080, '', '', 'Local', 'inactive', 'Dev tunnel'),
  (4, 'APAC Edge', 'http', 'apac.proxy.dorian', 8080, 'dorian', 'secret', 'SG', 'error', 'Intermittent timeouts');

-- Keep AUTO_INCREMENT in sync after explicit IDs
ALTER TABLE `proxies` AUTO_INCREMENT = 5;

-- Seed gmails
INSERT INTO `gmails` (`id`, `email`, `password`, `app_password`, `recovery_email`, `recovery_phone`, `twofa_secret`, `backup_codes`, `label`, `status`, `notes`) VALUES
  (1, 'jane@dorian.center', 'gmail-pass-1', 'app-pass-1', 'jane.recovery@dorian.center', '', 'JBSWY3DPEHPK3PXP', '1234-5678\n9012-3456\n5678-9012', 'Jane Ortiz', 'active', 'Primary mailbox for Twitter'),
  (2, 'dan@dorian.center', 'gmail-pass-2', 'app-pass-2', '', '', '', '', 'Dan Kim', 'error', 'Login challenges'),
  (3, 'priya@dorian.center', 'gmail-pass-3', 'app-pass-3', 'priya.backup@dorian.center', '+1 555 0100', 'KRSXG5CTMVRXEZLU', '1111-2222\n3333-4444', 'Priya Shah', 'active', 'LinkedIn recovery mailbox'),
  (4, 'alex@dorian.center', 'gmail-pass-4', 'app-pass-4', '', '', '', '', 'Alex Reed', 'inactive', 'Paused TikTok mailbox');

ALTER TABLE `gmails` AUTO_INCREMENT = 5;

-- Seed accounts
INSERT INTO `accounts` (
  `platform`, `first_name`, `last_name`, `password`, `email`, `email_password`,
  `birthday`, `gender`, `proxy_mode`, `proxy_id`, `status`, `connected_by`, `connected_at`
) VALUES
  ('twitter', 'Jane', 'Ortiz', 'secret', 'jane@dorian.center', 'app-pass-1',
   '1992-04-12', 'female', 'manual', 1, 'active', 'Jane Ortiz', '2026-01-12'),
  ('instagram', 'Dan', 'Kim', 'secret', 'dan@dorian.center', 'app-pass-2',
   '1988-11-03', 'male', 'none', NULL, 'error', 'Dan Kim', '2026-02-03'),
  ('linkedin', 'Priya', 'Shah', 'secret', 'priya@dorian.center', 'app-pass-3',
   '1995-07-21', 'female', 'auto', 2, 'active', 'Jane Ortiz', '2026-03-21'),
  ('tiktok', 'Alex', 'Reed', 'secret', 'alex@dorian.center', 'app-pass-4',
   '1999-01-30', 'other', 'manual', 3, 'expired', 'Priya Shah', '2025-11-08');
