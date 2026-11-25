INSERT INTO FUNCTION s3(
    {url:String},
    {access_key:String},
    {secret_key:String},
    'Parquet'
)
SELECT
    id,
    service,
    created_at,
    action_type,
    entity_type,
    realm,
    actor,
    ip,
    user_agent,
    input
FROM
    s3(
        {previous_url:String},
        {access_key:String},
        {secret_key:String},
        'Parquet'
    )
UNION ALL
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
AND
    toString(id) NOT IN (
        SELECT
            id
        FROM
            s3(
                {previous_url:String},
                {access_key:String},
                {secret_key:String},
                'Parquet'
            )
    )
