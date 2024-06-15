-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS tinyr 
  WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : '1' };

-- Short table
CREATE TABLE IF NOT EXISTS tinyr.short (
  short text,
  long text,
  owner bigint,
  PRIMARY KEY (short)
);

-- Users table
CREATE TABLE IF NOT EXISTS tinyr.users (
  uid bigint,
  name text,
  email text,
  PRIMARY KEY (uid)
);
