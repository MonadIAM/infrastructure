package archive

const partitionsQuery = `
SELECT DISTINCT partition
FROM system.parts
WHERE database = 'audit'
  AND table = {table:String}
  AND active
  AND partition <= toString(toYYYYMM(subtractDays(now(), {after_days:UInt64})))
ORDER BY partition
`

const loggedPartitionQuery = `
SELECT count() FROM audit.archive_log
WHERE table_name = {table:String} AND partition = {partition:String}
`

const sourceCountQuery = `
SELECT count() FROM audit.{table:Identifier}
WHERE toYYYYMM(created_at) = {partition:UInt32}
`

const archiveCountQuery = `
SELECT count() FROM s3({url:String}, {access_key:String}, {secret_key:String}, 'Parquet')
`

const recordArchiveQuery = `
INSERT INTO audit.archive_log (table_name, partition, row_count, archived_at)
SELECT {table:String}, {partition:String}, {row_count:UInt64}, now64(3)
`

const exportQuery = `
INSERT INTO FUNCTION s3({url:String}, {access_key:String}, {secret_key:String}, 'Parquet')
SELECT %s
FROM audit.{table:Identifier}
WHERE toYYYYMM(created_at) = {partition:UInt32}
`
