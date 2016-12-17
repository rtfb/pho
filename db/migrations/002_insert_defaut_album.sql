-- +migrate Up
create unique index ix_album_name on album (name);
create unique index ix_album_url on album (url);
insert into album (name, url) values ('default', 'default');

-- +migrate Down
delete from album where name = 'default';
drop index ix_album_url;
drop index ix_album_name;
