INSERT INTO FUNCTION s3(
    {url:String},
    {access_key:String},
    {secret_key:String},
    'Parquet'
)
SELECT
    toString(id) AS id,
    service,
    created_at,
    action_type,
    entity_type,
    toString(realm) AS realm,
    toString(actor) AS actor,
    ip,
    user_agent,
    input
FROM
    audit.audit_log
WHERE
    toYYYYMM(created_at) = {partition:UInt32}
