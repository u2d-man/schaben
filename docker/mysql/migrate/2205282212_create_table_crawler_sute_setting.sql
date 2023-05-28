CREATE TABLE crawler_site_setting
(
    id int auto_increment PRIMARY KEY,
    crawler_site_id int not null,
    title varchar(100) not null,
    body varchar(100) not null,
    article_update_date varchar(100) not null,
    created_at datetime default CURRENT_TIMESTAMP,
    updated_at datetime default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    foreign key fk_crawler_site_id(crawler_site_id) references crawler_site(id)
)