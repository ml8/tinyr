CREATE DATABASE IF NOT EXISTS tinyr;
USE tinyr;

-- Short table
CREATE TABLE IF NOT EXISTS shorts (
  short_url VARCHAR(512) NOT NULL,
  long_url VARCHAR(2048) NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (short_url)
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
  user_id BIGINT UNSIGNED NOT NULL,
  email VARCHAR(1024) NOT NULL,
  name VARCHAR(1024),
  PRIMARY KEY (user_id)
);
