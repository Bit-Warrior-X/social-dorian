-- Incremental migration: add password, twofa_secret, backup_codes to gmails table
USE `doriansocial`;

ALTER TABLE `gmails`
  ADD COLUMN IF NOT EXISTS `password`      VARCHAR(255) NOT NULL DEFAULT '' AFTER `email`,
  ADD COLUMN IF NOT EXISTS `twofa_secret`  VARCHAR(255) NOT NULL DEFAULT '' AFTER `recovery_phone`,
  ADD COLUMN IF NOT EXISTS `backup_codes`  TEXT         NOT NULL AFTER `twofa_secret`;
