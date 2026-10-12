package archive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Job struct {
	settings Settings
	client   *clickhouse.Client
	storage  *minio.Client
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
	job.client = clickhouse.New(querySettings, true)
	job.storage, err = seaweedfs.New(job.settings.Storage).Client(false)
	if err != nil {
		return err
	}

	sources := job.tables()

	for _, source := range sources {
		err := job.archiveTable(ctx, source)
		if err != nil {
			return err
		}
	}

	return nil
}

func (job *Job) Loop(ctx context.Context) error {
	err := job.settings.Validate()
	if err != nil {
		return err
	}

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		err = job.Run(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			_, writeErr := fmt.Fprintf(job.output, "Archive cycle failed: %v; retrying next cycle.\n", err)
			if writeErr != nil {
				return writeErr
			}
		}

		timer := time.NewTimer(job.settings.Interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (job *Job) archiveTable(ctx context.Context, source table) error {
	parameters := map[string]string{
		"table":      source.Name,
		"after_days": strconv.Itoa(job.settings.AfterDays),
	}
	partitions, err := job.client.Query(ctx, partitionsQuery, parameters)
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
	logged, err := job.count(ctx, loggedPartitionQuery, parameters)
	if err != nil {
		return err
	}

	if logged > 0 {
		return nil
	}

	rows, err := job.count(ctx, sourceCountQuery, parameters)
	if err != nil {
		return err
	}

	key := partition + ".parquet"
	archiveParameters := map[string]string{
		"table":      source.Name,
		"partition":  partition,
		"url":        strings.TrimRight(job.settings.Storage.Endpoint, "/") + "/" + source.Bucket + "/" + key,
		"access_key": job.settings.Storage.ArchiverKey,
		"secret_key": job.settings.Storage.ArchiverSecret,
	}
	err = job.exportMissing(ctx, source, key, archiveParameters)
	if err != nil {
		return err
	}

	exported, err := job.count(ctx, archiveCountQuery, archiveParameters)
	if err != nil {
		return err
	}

	current, err := job.count(ctx, sourceCountQuery, parameters)
	if err != nil {
		return err
	}

	if exported != rows || current != rows {
		return errors.New("archive row count mismatch; archive is not marked complete")
	}

	parameters["row_count"] = strconv.FormatUint(exported, 10)
	_, err = job.client.Query(ctx, recordArchiveQuery, parameters)
	return err
}

func (job *Job) exportMissing(ctx context.Context, source table, key string, parameters map[string]string) error {
	_, err := job.storage.StatObject(ctx, source.Bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return nil
	}

	code := minio.ToErrorResponse(err).Code
	if code != "NoSuchKey" && code != "NotFound" {
		return errors.New("cannot inspect archive object")
	}

	query := fmt.Sprintf(exportQuery, source.Projection)
	_, err = job.client.Query(ctx, query, parameters)
	return err
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
	settings := Settings{Timeout: 30 * time.Minute}
	err := environment.Decode(&settings)
	if err != nil {
		return settings, err
	}

	settings.ClickHouse, err = clickhouse.Load(environment)
	if err != nil {
		return settings, err
	}

	settings.Storage, err = seaweedfs.Load(environment)
	if err != nil {
		return settings, err
	}

	seconds, err := environment.PositiveInt("ARCHIVE_INTERVAL_SECONDS")
	if err != nil {
		return settings, err
	}

	if seconds > int((24*time.Hour*365)/time.Second) {
		return settings, errors.New("archive interval must not exceed one year")
	}

	settings.Interval = time.Duration(seconds) * time.Second
	return settings, settings.Validate()
}
