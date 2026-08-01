create table if not exists transactionrows(
    uuid uuid primary key default gen_random_uuid(),
    user_uuid uuid not null references users(uuid) on delete cascade,
    amount_e5 bigint not null,
    country_iso2 char(2) not null,
    category varchar(255) not null,
    txn_type varchar(255) not null,
    bank_name varchar(255) not null,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);