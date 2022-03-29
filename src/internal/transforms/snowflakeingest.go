package transforms

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/sirupsen/logrus"
)

const fileMode = 0755

type SnowflakeStageFilesParams struct {
	// Instrumentation
	Logger *logrus.Logger

	// PFS
	InputDir, OutputDir string

	// Snowflake
	Query         string
	InternalStage string
	PartitionBy   string
	FileFormat    string // CSV | JSON | PARQUET
	MaxFileSize   uint   // default 16777216
}

func SnowflakeStageFiles(ctx context.Context, db *sqlx.DB, params SnowflakeStageFilesParams) error {
	log := params.Logger

	// COPY INTO <location>
	copy := fmt.Sprintf(`COPY INTO @%s
		FROM (%s)
		PARTITION BY (%s)
		FILE_FORMAT = (TYPE = %s)
		MAX_FILE_SIZE = %d
	`, params.InternalStage, params.Query, params.PartitionBy, params.FileFormat, params.MaxFileSize)
	log.Infof("Executing query: %s", copy)
	_, err := db.Exec(copy)
	if err != nil {
		return errors.EnsureStack(err)
	}

	// LIST <location>
	list := fmt.Sprintf("list @%s", params.InternalStage)
	log.Infof("Executing query: %s", list)
	rows, err := db.Query(list)
	if err != nil {
		return errors.EnsureStack(err)
	}
	defer rows.Close()
	files := []string{}
	for rows.Next() {
		var (
			name, md5, last_modified string
			size                     int
		)
		if err = rows.Scan(&name, &size, &md5, &last_modified); err != nil {
			return errors.EnsureStack(err)
		}
		files = append(files, name)
	}
	rows.Close()
	if len(files) == 0 {
		log.Info("Zero files exported")
		return nil
	}

	// write file names from Snowflake to OutputDir
	timestamp, err := readCronTimestamp(params.Logger, params.InputDir)
	if err != nil {
		return err
	}
	contents := fmt.Sprintf("%d\n", timestamp)
	// Snowflake file hiearchy can have arbitrary depth, we need to re-create this locally
	for _, f := range files {
		parent := filepath.Dir(f)
		if err = os.MkdirAll(filepath.Join(params.OutputDir, parent), fileMode); err != nil {
			return errors.EnsureStack(err)
		}
		outputPath := filepath.Join(params.OutputDir, f)
		if err = ioutil.WriteFile(outputPath, []byte(contents), fileMode); err != nil {
			return errors.EnsureStack(err)
		}
	}
	return nil
}

func Get(ctx context.Context, db *sqlx.DB) {}
