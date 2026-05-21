CREATE TABLE monitors (
    monitor_id varchar(36) PRIMARY KEY,
    url varchar(2048) NOT NULL UNIQUE, -- tough luck if your url is any longer
    website_name varchar(50) NOT NULL,
    website_description varchar(500),
    created_at TIMESTAMPTZ NOT NULL,
    first_created_by varchar(36),
    FOREIGN KEY (first_created_by) REFERENCES
                      users(user_id)
);
