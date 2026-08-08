CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

create table if not exists envelope_group (
    id uuid primary key default gen_random_uuid(),
    user_uuid uuid not null,
    name varchar(50) not null,
    is_system boolean default false,
    created_at timestamptz default now(),
    updated_at timestamptz default now(),
    foreign key (user_uuid) references users(uuid) on delete cascade
);

create index if not exists idx_envelope_group_user_uuid on envelope_group(user_uuid);
create index if not exists idx_envelope_group_id on envelope_group(id);

create table if not exists envelope (
    id uuid primary key default gen_random_uuid(),
    user_uuid uuid not null,
    envelope_group_id uuid not null,
    is_system boolean default false,
    target_amount_e5 bigint default 0,
    cadence varchar(50) default 'monthly',
    country_iso varchar(10) not null default 'IN',
    created_at timestamptz default now(),
    updated_at timestamptz default now(),
    foreign key (user_uuid) references users(uuid) on delete cascade,
    foreign key (envelope_group_id) references envelope_group(id) on delete cascade
);

create index if not exists idx_envelope_user_uuid on envelope(user_uuid);
create index if not exists idx_envelope_id on envelope(id);
create index if not exists idx_envelope_group_id on envelope(envelope_group_id);

create table if not exists allocation (
    id uuid primary key default gen_random_uuid(),
    envelope_id uuid not null,
    start_date date not null,
    end_date date not null,
    allocated_amount_e5 bigint not null,
    created_at timestamptz default now(),
    updated_at timestamptz default now(),
    foreign key (envelope_id) references envelope(id) on delete cascade,
    constraint no_overlapping_allocations
        exclude using gist (
            envelope_id WITH =, 
            daterange(start_date, end_date, '[]') WITH &&
        )
);


create index if not exists idx_allocation_id on allocation(id);
create index if not exists idx_allocation_envelope_id on allocation(envelope_id);