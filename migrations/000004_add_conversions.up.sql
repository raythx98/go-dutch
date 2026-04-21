create table if not exists exchange_rate_snapshots
(
    id                 bigserial primary key,
    base_currency_code varchar(3)                               not null,
    rates              jsonb                                    not null,
    fetched_at         timestamp default timezone('UTC', now()) not null,
    unique (base_currency_code)
);

create table if not exists conversions
(
    id                bigserial primary key,
    source_expense_id bigint                                   not null,
    target_expense_id bigint                                   not null,
    rate              decimal(20, 6)                           not null,
    created_at        timestamp default timezone('UTC', now()) not null,
    unique (source_expense_id),
    unique (target_expense_id),
    constraint fk_source_expense_id foreign key (source_expense_id) references expenses (id)
        on delete cascade on update cascade,
    constraint fk_target_expense_id foreign key (target_expense_id) references expenses (id)
        on delete cascade on update cascade
);

create index idx_conversions_target_expense_id on conversions (target_expense_id);
