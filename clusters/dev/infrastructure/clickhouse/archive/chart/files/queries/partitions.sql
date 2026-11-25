SELECT DISTINCT
    partition
FROM
    system.parts
WHERE
    database = 'audit'
AND
    table = {table:String}
AND
    active
AND
    partition < toString(toYYYYMM(subtractDays(now(), {after_days:UInt64})))
ORDER BY
    partition
