CREATE TYPE summary_status AS ENUM('Down','Struggling','Working');

CREATE TABLE checks (
    check_id varchar(36) PRIMARY KEY,
    monitor_id varchar(36) NOT NULL,
    recorded TIMESTAMPTZ NOT NULL,
    status summary_status NOT NULL,
    reports int NOT NULL,
    FOREIGN KEY (monitor_id) REFERENCES monitors(monitor_id)
);