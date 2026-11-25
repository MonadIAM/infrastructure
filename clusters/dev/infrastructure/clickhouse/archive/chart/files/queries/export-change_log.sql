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
    toString(audit_entry) AS audit_entry,
    change_type,
    entity_type,
    toString(entity) AS entity,
    delta
FROM
    audit.change_log
WHERE
    toYYYYMM(created_at) = {partition:UInt32}
