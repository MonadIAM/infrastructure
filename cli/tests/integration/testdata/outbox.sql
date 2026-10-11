CREATE SCHEMA system;
CREATE TABLE system.outbox (
    id uuid PRIMARY KEY,
    destination_topic text NOT NULL,
    action_type text NOT NULL,
    payload jsonb NOT NULL
);
