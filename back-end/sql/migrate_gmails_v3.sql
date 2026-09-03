-- Incremental migration: add pin_code to gmails table
USE `doriansocial`;

ALTER TABLE `gmails`
  ADD COLUMN IF NOT EXISTS `pin_code` VARCHAR(16) NOT NULL DEFAULT '' AFTER `backup_codes`;

