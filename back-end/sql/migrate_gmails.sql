-- Incremental migration for existing databases
USE `doriansocial`;

CREATE TABLE IF NOT EXISTS `gmails` (
  `id`              INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `email`           VARCHAR(255) NOT NULL,
  `app_password`    VARCHAR(255) NOT NULL,
  `recovery_email`  VARCHAR(255) NOT NULL DEFAULT '',
  `recovery_phone`  VARCHAR(60)  NOT NULL DEFAULT '',
  `label`           VARCHAR(120) NOT NULL DEFAULT '',
  `status`          ENUM('active', 'inactive', 'error') NOT NULL DEFAULT 'active',
  `notes`           TEXT         NOT NULL,
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gmails_email` (`email`),
  KEY `idx_gmails_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `gmails` (`email`, `app_password`, `label`, `status`, `notes`)
SELECT DISTINCT a.email, COALESCE(NULLIF(a.email_password, ''), 'changeme'),
       TRIM(CONCAT(a.first_name, ' ', a.last_name)), 'active', 'Imported from social account'
FROM `accounts` a
WHERE a.email <> '';
