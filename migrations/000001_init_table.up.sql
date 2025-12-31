create table if not exists users
(
    id         bigserial primary key,
    username   varchar(20) unique                       not null,
    email      varchar(255) unique                      not null,
    password   varchar(255)                             not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null
);

create table if not exists groups
(
    id         bigserial primary key,
    name       varchar(20)                              not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null
);

create table if not exists user_group
(
    id         bigserial primary key,
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

create table if not exists currencies
(
    id         bigserial primary key,
    code       varchar(3)                               not null,
    name       varchar(50)                              not null,
    symbol     varchar(5)                               not null,
    created_at timestamp default timezone('UTC', now()) not null,
    is_deleted boolean   default false                  not null
);

create table if not exists expenses
(
    id          bigserial primary key,
    group_id    bigint                                   not null,
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

create table if not exists expense_payers
(
    id         bigserial primary key,
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
    id         bigserial primary key,
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