FROM mysql:8.0

COPY ./docker/mysql/my.cnf /etc/mysql/conf.d/

ENV LANG ja_JP.UTF-8
CMD ["mysqld", "--character-set-server=utf8", "--collation-server=utf8_unicode_ci"]
