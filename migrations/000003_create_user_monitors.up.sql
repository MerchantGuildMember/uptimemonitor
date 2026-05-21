CREATE TABLE user_monitors (
    user_id varchar(36) NOT NULL,
    monitor_id varchar(36) NOT NULL,
    PRIMARY KEY (user_id, monitor_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (monitor_id) REFERENCES monitors(monitor_id)
);