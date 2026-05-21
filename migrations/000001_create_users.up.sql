CREATE TABLE users (
    user_id varchar(36) PRIMARY KEY,
    username varchar(20) NOT NULL UNIQUE,
    email varchar(50) NOT NULL UNIQUE,
    password char(60) NOT NULL
);