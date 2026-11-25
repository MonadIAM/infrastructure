package archive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Job struct {
	settings Settings
	client   *clickhouse.Client
	queries  map[string]string
	output   io.Writer
}

var partitionPattern = regexp.MustCompile(`^[0-9]{6}$`)

func New(settings Settings, output io.Writer) *Job {
	return &Job{settings: settings, output: output}
}

func (job *Job) Run(ctx context.Context) error {
	err := job.settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, job.settings.Timeout)
	defer cancel()

	querySettings := job.settings.ClickHouse
	querySettings.Timeout = job.settings.Timeout
	job.client = clickhouse.New(querySettings)

	sources := job.tables()
	job.queries, err = loadQueries(job.settings.QueryDir, sources)
	if err != nil {
		return err
	}

	for _, source := range sources {
		err := job.archiveTable(ctx, source)
		if err != nil {
			return err
		}
	}

	return nil
}

func (job *Job) archiveTable(ctx context.Context, source table) error {
	parameters := map[string]string{
		"table":      source.Name,
		"after_days": strconv.Itoa(job.settings.AfterDays),
	}
	partitions, err := job.client.Query(ctx, job.queries["partitions"], parameters)
	if err != nil {
		return err
	}

	for _, partition := range strings.Fields(partitions) {
		if !partitionPattern.MatchString(partition) {
			return errors.New("unexpected archive partition")
		}

		err := job.archivePartition(ctx, source, partition)
		if err != nil {
			return fmt.Errorf("archive.Job.archiveTable: archive %s/%s: %w", source.Name, partition, err)
		}

		_, err = fmt.Fprintf(job.output, "Verified archive %s/%s.\n", source.Name, partition)
		if err != nil {
			return err
		}
	}

	return nil
}

func (job *Job) archivePartition(ctx context.Context, source table, partition string) error {
	parameters := map[string]string{
		"table":     source.Name,
		"partition": partition,
	}
	state, err := job.readPartitionState(ctx, source, parameters)
	if err != nil {
		return err
	}

	if state.exists && state.missing == 0 {
		return nil
	}

	key, err := revisionKey(partition)
	if err != nil {
		return err
	}

	archiveParameters := job.objectParameters(source, key)
	for name, value := range parameters {
		archiveParameters[name] = value
	}

	rows, err := job.exportPartition(ctx, source, state, parameters, archiveParameters)
	if err != nil {
		return err
	}

	exported, err := job.verifyPartition(ctx, state, rows, parameters, archiveParameters)
	if err != nil {
		return err
	}

	parameters["row_count"] = strconv.FormatUint(exported, 10)
	parameters["object_key"] = key
	_, err = job.client.Query(ctx, job.queries["record"], parameters)
	return err
}

func (job *Job) readPartitionState(ctx context.Context, source table, parameters map[string]string) (partitionState, error) {
	latest, err := job.client.Query(ctx, job.queries["latest"], parameters)
	if err != nil {
		return partitionState{}, err
	}

	if latest == "" {
		return partitionState{}, nil
	}

	fields := strings.Split(latest, "|")
	if len(fields) != 2 || fields[0] == "" {
		return partitionState{}, errors.New("invalid archive log entry")
	}

	archived, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return partitionState{}, errors.New("invalid archived row count")
	}

	previous := job.objectParameters(source, fields[0])
	actual, err := job.count(ctx, job.queries["archive-count"], previous)
	if err != nil {
		return partitionState{}, err
	}

	if actual != archived {
		return partitionState{}, errors.New("logged archive row count does not match Parquet object")
	}

	parameters["previous_url"] = previous["url"]
	parameters["access_key"] = previous["access_key"]
	parameters["secret_key"] = previous["secret_key"]
	missing, err := job.count(ctx, job.queries["missing-count"], parameters)
	if err != nil {
		return partitionState{}, err
	}

	return partitionState{exists: true, archived: archived, missing: missing}, nil
}

func (job *Job) exportPartition(ctx context.Context, source table, state partitionState, parameters, archiveParameters map[string]string) (uint64, error) {
	rows := state.archived + state.missing
	query := job.queries["merge-"+source.Name]
	if !state.exists {
		var err error
		rows, err = job.count(ctx, job.queries["source-count"], parameters)
		if err != nil {
			return 0, err
		}

		query = job.queries["export-"+source.Name]
	}

	_, err := job.client.Query(ctx, query, archiveParameters)
	if err != nil {
		return 0, err
	}

	return rows, nil
}

func (job *Job) verifyPartition(ctx context.Context, state partitionState, rows uint64, parameters, archiveParameters map[string]string) (uint64, error) {
	exported, err := job.count(ctx, job.queries["archive-count"], archiveParameters)
	if err != nil {
		return 0, err
	}

	if exported != rows {
		return 0, errors.New("archive row count mismatch; archive is not marked complete")
	}

	query := job.queries["missing-count"]
	expected := state.missing
	if !state.exists {
		query = job.queries["source-count"]
		expected = rows
	}

	current, err := job.count(ctx, query, parameters)
	if err != nil {
		return 0, err
	}

	if current != expected {
		return 0, errors.New("source partition changed during archive export")
	}

	return exported, nil
}

func (job *Job) objectParameters(source table, key string) map[string]string {
	return map[string]string{
		"url":        strings.TrimRight(job.settings.Storage.Endpoint, "/") + "/" + source.Bucket + "/" + key,
		"access_key": job.settings.Storage.ArchiverKey,
		"secret_key": job.settings.Storage.ArchiverSecret,
	}
}

func revisionKey(partition string) (string, error) {
	var suffix [16]byte
	_, err := rand.Read(suffix[:])
	if err != nil {
		return "", err
	}

	return partition + "-" + hex.EncodeToString(suffix[:]) + ".parquet", nil
}

func (job *Job) count(ctx context.Context, query string, parameters map[string]string) (uint64, error) {
	value, err := job.client.Query(ctx, query, parameters)
	if err != nil {
		return 0, err
	}

	result, err := strconv.ParseUint(value, 10, 64)
	if err == nil {
		return result, nil
	} else {
		return 0, errors.New("invalid ClickHouse row count")
	}
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{}
	var err error
	settings.Timeout, err = environment.Duration("ARCHIVE_TIMEOUT", "30m")
	if err != nil {
		return settings, err
	}

	settings.AfterDays, err = environment.PositiveInt("ARCHIVE_AFTER_DAYS")
	if err != nil {
		return settings, err
	}

	settings.ClickHouse, err = clickhouse.Load(environment)
	if err != nil {
		return settings, err
	}

	settings.Storage = seaweedfs.Settings{
		Endpoint:       environment["SEAWEEDFS_ENDPOINT"],
		Region:         environment.Get("SEAWEEDFS_REGION", "us-east-1"),
		ArchiverKey:    environment["SEAWEEDFS_ARCHIVER_ACCESS_KEY"],
		ArchiverSecret: environment["SEAWEEDFS_ARCHIVER_SECRET_KEY"],
		Buckets: []seaweedfs.Bucket{
			{Name: environment["SEAWEEDFS_AUDIT_LOG_BUCKET"]},
			{Name: environment["SEAWEEDFS_CHANGE_LOG_BUCKET"]},
		},
	}
	settings.QueryDir = environment.Get("CLICKHOUSE_ARCHIVE_QUERY_DIR", "/queries")
	return settings, settings.Validate()
}
