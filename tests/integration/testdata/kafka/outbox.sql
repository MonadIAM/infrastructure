CREATE SCHEMA IF NOT EXISTS system;

CREATE TABLE IF NOT EXISTS system.outbox (
    id uuid PRIMARY KEY,
    sequence_number bigserial,
    action_type varchar(64) NOT NULL,
    destination_topic varchar(128) NOT NULL,
    payload jsonb NOT NULL,
    metadata jsonb,
    created_at timestamptz(3) NOT NULL
);
