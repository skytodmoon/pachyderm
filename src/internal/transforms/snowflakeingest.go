package transforms

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

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
	Compression   string
	MaxFileSize   uint // default 16777216
}

type SnowflakeGetParams struct {
	Logger *logrus.Logger

	// PFS
	InputDir, OutputDir string

	// Snowflake
	Parallel int // TODO
}

// SnowflakeStageFiles exports files to a Snowfalke internal stage, then writes the file names to a local directory.
// The content of each file is based on the cron timestamp in the InputDir.
func SnowflakeStageFiles(ctx context.Context, db *sqlx.DB, params SnowflakeStageFilesParams) error {
	log := params.Logger

	// COPY INTO <location>
	copy := fmt.Sprintf(`COPY INTO @%s
		FROM (%s)
		PARTITION BY (%s)
		FILE_FORMAT = (TYPE = %s COMPRESSION = %s)
		MAX_FILE_SIZE = %d
	`, params.InternalStage, params.Query, params.PartitionBy, params.FileFormat, params.Compression, params.MaxFileSize)
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
	for partition, filename := range files {
		outputPath := filepath.Join(params.OutputDir, strconv.Itoa(partition))
		contents := filepath.Join(params.InternalStage, filename)
		if err = ioutil.WriteFile(outputPath, []byte(contents), fileMode); err != nil {
			return errors.EnsureStack(err)
		}
	}
	return nil
}

func SnowflakeGet(ctx context.Context, db *sqlx.DB, params SnowflakeGetParams) error {
	log := params.Logger

	return bijectiveMap(params.InputDir, params.OutputDir, IdentityPM, func(r io.Reader, w io.Writer) error {
		snowflakePath, err := io.ReadAll(r)
		if err != nil {
			return errors.EnsureStack(err)
		}

		// use a temp directory
		tempDirname, err := os.MkdirTemp("", "snowflake-get")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDirname)
		query := fmt.Sprintf("GET @%s file://%s overwrite=true", snowflakePath, tempDirname)
		log.Infof("Executing: %s", query)
		rows, err := db.Query(query)
		if err != nil {
			return errors.EnsureStack(err)
		}
		defer rows.Close()

		files := []string{}
		for rows.Next() {
			var (
				file, size, status, message string
			)
			if err = rows.Scan(&file, &size, &status, &message); err != nil {
				return errors.EnsureStack(err)
			}
			files = append(files, file)
			log.Infof("%s, %s, %s, %s", file, size, status, message)
		}
		rows.Close()

		for _, f := range files {
			bytes, err := ioutil.ReadFile(filepath.Join(tempDirname, f))
			if err != nil {
				return err
			}
			w.Write(bytes)
		}
		return nil
	})
}
