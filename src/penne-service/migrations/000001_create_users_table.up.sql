create table if not exists users (
    uuid uuid primary key default gen_random_uuid(),
    name varchar(255) not null unique,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);