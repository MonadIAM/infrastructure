SELECT
    count()
FROM
    audit.{table:Identifier}
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
