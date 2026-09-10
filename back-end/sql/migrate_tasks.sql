-- Tasks / campaigns engine
CREATE TABLE IF NOT EXISTS `tasks` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `type` ENUM('report','post','browse','login_test') NOT NULL,
  `title` VARCHAR(255) NOT NULL DEFAULT '',
  `target_url` TEXT NOT NULL,
  `status` ENUM('pending','queued','running','completed','failed','cancelled') NOT NULL DEFAULT 'pending',
  `delay_min_sec` INT UNSIGNED NOT NULL DEFAULT 5,
  `delay_max_sec` INT UNSIGNED NOT NULL DEFAULT 15,
  `use_account_proxy` TINYINT(1) NOT NULL DEFAULT 1,
  `created_by` VARCHAR(120) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `started_at` DATETIME NULL,
  `finished_at` DATETIME NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tasks_status` (`status`),
  KEY `idx_tasks_type` (`type`),
  KEY `idx_tasks_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_items` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` INT UNSIGNED NOT NULL,
  `account_id` INT UNSIGNED NOT NULL,
  `status` ENUM('pending','running','success','failed','cancelled') NOT NULL DEFAULT 'pending',
  `message` TEXT NOT NULL,
  `started_at` DATETIME NULL,
  `finished_at` DATETIME NULL,
  PRIMARY KEY (`id`),
  KEY `idx_task_items_task` (`task_id`),
  KEY `idx_task_items_account` (`account_id`),
  KEY `idx_task_items_status` (`status`),
  CONSTRAINT `fk_task_items_task` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_task_items_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_logs` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` INT UNSIGNED NOT NULL,
  `account_id` INT UNSIGNED NULL,
  `level` ENUM('info','success','warn','error') NOT NULL DEFAULT 'info',
  `message` TEXT NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_task_logs_task` (`task_id`),
  CONSTRAINT `fk_task_logs_task` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `workspace_credits` (
  `id` INT UNSIGNED NOT NULL PRIMARY KEY,
  `balance` INT NOT NULL DEFAULT 0,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `workspace_credits` (`id`, `balance`) VALUES (1, 1000);
