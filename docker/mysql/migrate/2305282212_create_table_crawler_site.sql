CREATE TABLE crawler_site
(
    id         int auto_increment PRIMARY KEY,
    domain     varchar(100) not null,
    url        varchar(100) not null,
    created_at datetime default CURRENT_TIMESTAMP,
    updated_at datetime default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)