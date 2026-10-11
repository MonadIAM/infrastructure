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

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

var partitionPattern = regexp.MustCompile(`^[0-9]{6}$`)

func Run(ctx context.Context, settings Settings, output io.Writer) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	client := clickhouse.Client(settings.ClickHouse, true)
	client.HTTP.Timeout = settings.Timeout
	storage, err := seaweedfs.Client(settings.Storage, false)
	if err != nil {
		return err
	}

	for _, source := range tableDefinitions(settings) {
		err := archiveTable(ctx, client, storage, settings, source, output)
		if err != nil {
			return err
		}
	}

	return nil
}

func archiveTable(ctx context.Context, client infrahttp.Client, storage *minio.Client, settings Settings, source table, output io.Writer) error {
	query := fmt.Sprintf("SELECT DISTINCT partition FROM system.parts WHERE database = 'audit' AND table = '%s' AND active AND partition <= toString(toYYYYMM(now() - INTERVAL %d DAY)) ORDER BY partition", source.Name, settings.AfterDays)
	partitions, err := clickhouse.Query(ctx, client, query)
	if err != nil {
		return err
	}

	for _, partition := range strings.Fields(partitions) {
		if !partitionPattern.MatchString(partition) {
			return errors.New("unexpected archive partition")
		}

		err := archivePartition(ctx, client, storage, settings, source, partition)
		if err != nil {
			return fmt.Errorf("archive %s/%s: %w", source.Name, partition, err)
		}

		_, err = fmt.Fprintf(output, "Verified archive %s/%s.\n", source.Name, partition)
		if err != nil {
			return err
		}
	}

	return nil
}

func count(ctx context.Context, client infrahttp.Client, query string) (uint64, error) {
	value, err := clickhouse.Query(ctx, client, query)
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

func archivePartition(ctx context.Context, client infrahttp.Client, storage *minio.Client, settings Settings, source table, partition string) error {
	filter := fmt.Sprintf("table_name = '%s' AND partition = '%s'", source.Name, partition)
	logged, err := count(ctx, client, "SELECT count() FROM audit.archive_log WHERE "+filter)
	if err != nil {
		return err
	}

	if logged > 0 {
		return nil
	}

	sourceQuery := fmt.Sprintf("SELECT count() FROM audit.%s WHERE toYYYYMM(created_at) = %s", source.Name, partition)
	rows, err := count(ctx, client, sourceQuery)
	if err != nil {
		return err
	}

	key := partition + ".parquet"
	address := strings.TrimRight(settings.Storage.Endpoint, "/") + "/" + source.Bucket + "/" + key
	function := "s3(" + clickhouse.Quote(address) + ", " + clickhouse.Quote(settings.Storage.ArchiverKey) + ", " + clickhouse.Quote(settings.Storage.ArchiverSecret) + ", 'Parquet')"
	err = exportMissing(ctx, client, storage, source, partition, key, function)
	if err != nil {
		return err
	}

	exported, err := count(ctx, client, "SELECT count() FROM "+function)
	if err != nil {
		return err
	}

	current, err := count(ctx, client, sourceQuery)
	if err != nil {
		return err
	}

	if exported != rows || current != rows {
		return errors.New("archive row count mismatch; archive is not marked complete")
	}

	query := fmt.Sprintf("INSERT INTO audit.archive_log (table_name, partition, row_count, archived_at) VALUES ('%s', '%s', %d, now64(3))", source.Name, partition, exported)
	_, err = clickhouse.Query(ctx, client, query)
	return err
}

func exportMissing(ctx context.Context, client infrahttp.Client, storage *minio.Client, source table, partition, key, function string) error {
	_, err := storage.StatObject(ctx, source.Bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return nil
	}

	code := minio.ToErrorResponse(err).Code
	if code != "NoSuchKey" && code != "NotFound" {
		return errors.New("cannot inspect archive object")
	}

	query := fmt.Sprintf("INSERT INTO FUNCTION %s SELECT %s FROM audit.%s WHERE toYYYYMM(created_at) = %s", function, source.Projection, source.Name, partition)
	_, err = clickhouse.Query(ctx, client, query)
	return err
}

func Loop(ctx context.Context, settings Settings, output io.Writer) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		err = Run(ctx, settings, output)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			_, writeErr := fmt.Fprintf(output, "Archive cycle failed: %v; retrying next cycle.\n", err)
			if writeErr != nil {
				return writeErr
			}
		}

		timer := time.NewTimer(settings.Interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
