create table if not exists users
(
    id bigserial primary key,
    username   varchar(20) unique                       not null,
    email      varchar(255) unique                      not null,
    password   varchar(255)                             not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null
);

create index idx_users_username_not_deleted on users (username) where
     is_deleted = false;
create index idx_users_email_not_deleted on users (email) where is_deleted =
     false;

create table if not exists groups
(
    id bigserial primary key,
    name         varchar(20)                              not null,
    invite_token varchar(20) unique                       not null,
    created_at   timestamp default timezone('UTC', now()) not null,
    is_deleted   boolean   default false                  not null
);

create index idx_groups_invite_token_not_deleted
    on groups (invite_token) where is_deleted = false;

create table if not exists user_group
(
    id bigserial primary key,
    user_id    bigint                                   not null,
    group_id   bigint                                   not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null,
    unique (user_id, group_id),
    constraint fk_user_id foreign key (user_id) references users (id)
        on delete cascade on update cascade,
    constraint fk_group_id foreign key (group_id) references groups (id)
        on delete cascade on update cascade
);

CREATE INDEX idx_user_group_group_id_not_deleted
    ON user_group (group_id, user_id) WHERE is_deleted = false;

CREATE INDEX idx_user_group_user_id_not_deleted
    ON user_group (user_id, group_id) WHERE is_deleted = false;

create table if not exists currencies
(
    id bigserial primary key,
    code       varchar(3) unique                        not null,
    name       varchar(50)                              not null,
    symbol     varchar(5)                               not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null
);

create index idx_currencies_not_deleted ON currencies (name) where is_deleted =
     false;

create table if not exists user_currency_preferences
(
    id bigserial primary key,
    user_id     bigint                                   not null,
    currency_id bigint                                   not null,
    use_count   int       default 0                      not null,
    created_at  timestamp default timezone('UTC', now()) not null,
    unique (user_id, currency_id),
    constraint fk_user_id foreign key (user_id) references users (id)
        on delete cascade on update cascade,
    constraint fk_currency_id foreign key (currency_id) references currencies (id)
        on delete cascade on update cascade
);

CREATE INDEX idx_ucp_user_ranking
    ON user_currency_preferences (user_id, currency_id, use_count);

create table if not exists expenses
(
    id bigserial primary key,
    group_id    bigint                                   not null,
    type        smallint                                 not null,
    name        varchar(100)                             not null,
    description varchar(1000)                            not null,
    amount      decimal(10, 2)                           not null,
    currency_id bigint                                   not null,
    expense_at  timestamp                                not null,
    created_at  timestamp default timezone('UTC', now()) not null,
    is_deleted  boolean   default false                  not null,
    constraint fk_group_id foreign key (group_id) references groups (id)
        on delete set null on update cascade,
    constraint fk_currency_id foreign key (currency_id) references currencies (id)
        on delete set null on update cascade
);

CREATE INDEX idx_expenses_group_order
    ON expenses (group_id, expense_at DESC) WHERE is_deleted = false;

create table if not exists expense_payers
(
    id bigserial primary key,
    expense_id bigint                                   not null,
    user_id    bigint                                   not null,
    amount     decimal(10, 2)                           not null,
    created_at timestamp default timezone('UTC', now()) not null,
    unique (expense_id, user_id),
    constraint fk_expense_id foreign key (expense_id) references expenses (id)
        on delete cascade on update cascade,
    constraint fk_user_id foreign key (user_id) references users (id)
        on delete cascade on update cascade
);

create table if not exists expense_shares
(
    id bigserial primary key,
    expense_id bigint                                   not null,
    user_id    bigint                                   not null,
    amount     decimal(10, 2)                           not null,
    created_at timestamp default timezone('UTC', now()) not null,
    unique (expense_id, user_id),
    constraint fk_expense_id foreign key (expense_id) references expenses (id)
        on delete cascade on update cascade,
    constraint fk_user_id foreign key (user_id) references users (id)
        on delete cascade on update cascade
);