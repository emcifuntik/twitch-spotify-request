-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

-- -----------------------------------------------------
-- Schema twspoty
-- -----------------------------------------------------
DROP SCHEMA IF EXISTS `twspoty` ;

-- -----------------------------------------------------
-- Schema twspoty
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `twspoty` DEFAULT CHARACTER SET utf8 ;
USE `twspoty` ;

-- -----------------------------------------------------
-- Table `twspoty`.`streamer`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `twspoty`.`streamer` (
  `streamer_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `streamer_channel_id` BIGINT NOT NULL,
  `streamer_name` VARCHAR(64) NOT NULL,
  `streamer_twitch_token` VARCHAR(256) NULL,
  `streamer_twitch_refresh` VARCHAR(256) NULL,
  `streamer_spotify_token` VARCHAR(256) NULL,
  `streamer_spotify_refresh` VARCHAR(256) NULL,
  `streamer_spotify_state` VARCHAR(64) NULL,
  PRIMARY KEY (`streamer_id`),
  UNIQUE INDEX `streamer_channel_id_UNIQUE` (`streamer_channel_id` ASC),
  UNIQUE INDEX `streamer_spotify_state_UNIQUE` (`streamer_spotify_state` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `twspoty`.`rewards`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `twspoty`.`rewards` (
  `reward_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reward_streamer` INT UNSIGNED NOT NULL,
  `reward_internal_id` TINYINT NOT NULL,
  `reward_twitch_id` VARCHAR(128) NULL,
  PRIMARY KEY (`reward_id`),
  INDEX `fk_rewards_streamer_idx` (`reward_streamer` ASC),
  CONSTRAINT `fk_rewards_streamer`
    FOREIGN KEY (`reward_streamer`)
    REFERENCES `twspoty`.`streamer` (`streamer_id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
