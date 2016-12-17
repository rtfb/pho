-- +migrate Up
create extension if not exists "uuid-ossp";
create table album (
    id uuid primary key default uuid_generate_v4(),
    name text,
    url text
);
create table stored_image (
    id uuid primary key default uuid_generate_v4(),
    upload_path text,
    orig_path text,
    thumb_path text,
    display_path text,
    uploaded_at timestamp,
    processed_at timestamp
);
create table image (
    id uuid primary key default uuid_generate_v4(),
    name text,
    url_name text,
    description text,
    store_id uuid not null references stored_image(id) on delete cascade on update cascade,
    album_id uuid not null references album(id) on delete cascade on update cascade
);

-- +migrate Down
drop table image;
drop table stored_image;
drop table album;
