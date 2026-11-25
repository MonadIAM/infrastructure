SELECT
    count()
FROM
    audit.{table:Identifier}
WHERE
    toYYYYMM(created_at) = {partition:UInt32}
