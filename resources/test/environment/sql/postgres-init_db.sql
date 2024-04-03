-- This file creates database goyav_db, a schema goyav_schema and a user goyav_user with password goyav_password --

CREATE DATABASE goyav_db;

\c goyav_db;


CREATE SCHEMA IF NOT EXISTS goyav_schema;

CREATE ROLE goyav_user WITH LOGIN PASSWORD 'goyav_password';

GRANT ALL PRIVILEGES ON SCHEMA goyav_schema TO goyav_user;

\q
