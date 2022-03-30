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

	// Pachyderm
	Name                string
	InputDir, OutputDir string

	// Snowflake
	Query         string
	InternalStage string
	PartitionBy   string
	FileFormat    string // CSV | JSON | PARQUET
	Compression   string // AUTO | GZIP | BZ2 | BROTLI | ZSTD | DEFLATE | RAW_DEFLATE | NONE
	MaxFileSize   uint   // default 16777216
}

type SnowflakeGetParams struct {
	Logger *logrus.Logger

	// PFS
	InputDir, OutputDir string

	// Snowflake
	Parallel int // TODO
}

// SnowflakeStageFiles exports data as files to a Snowflake internal stage, then writes the file names locally.
// Note: we name the output files by the partition number, and save the Snowflake filepath as the content.
// The consumer of these files will run Snowflake's GET command to download the file.
func SnowflakeStageFiles(ctx context.Context, db *sqlx.DB, params SnowflakeStageFilesParams) error {
	log := params.Logger

	// COPY INTO <location>
	timestamp, err := readCronTimestamp(params.Logger, params.InputDir)
	if err != nil {
		return err
	}
	// Namespace the files associated with this particular run by the name of the pipeline and cron timestamp.
	// This will be used by LIST to get only the files we need.
	snowflakePath := fmt.Sprintf("%s/%s/%d", params.InternalStage, params.Name, timestamp)
	copy := fmt.Sprintf(`COPY INTO %s
		FROM (%s)
		PARTITION BY (%s)
		FILE_FORMAT = (TYPE = %s COMPRESSION = %s)
		MAX_FILE_SIZE = %d
	`, snowflakePath, params.Query, params.PartitionBy, params.FileFormat, params.Compression, params.MaxFileSize)
	log.Infof("Query: %s", copy)
	_, err = db.Exec(copy)
	if err != nil {
		return errors.EnsureStack(err)
	}

	// LIST <location>
	list := fmt.Sprintf("LIST %s", snowflakePath)
	log.Infof("Query: %s", list)
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

	// write file names from Snowflake to OutputDir
	for partition, filename := range files {
		outputPath := filepath.Join(params.OutputDir, strconv.Itoa(partition))
		contents := filepath.Join(params.InternalStage, filename)
		if err = ioutil.WriteFile(outputPath, []byte(contents), fileMode); err != nil {
			return errors.EnsureStack(err)
		}
		log.Infof("Wrote %s to %s", contents, outputPath)
	}
	return nil
}

func SnowflakeGet(ctx context.Context, db *sqlx.DB, params SnowflakeGetParams) error {
	return bijectiveMap(params.InputDir, params.OutputDir, IdentityPM, func(r io.Reader, w io.Writer) error {
		log := params.Logger

		// src is a Snowflake path to a staged file
		// e.g. stage/a/b/c/file
		src, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		// create temp dir to save output of GET
		tempDir, err := os.MkdirTemp("", "snowflake-get")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		// GET @stage/a/b/c/file file:///localdir -> /localdir/file
		query := fmt.Sprintf("GET %s file://%s", src, tempDir)
		log.Infof("Query: %s", query)
		rows, err := db.Query(query)
		if err != nil {
			return errors.EnsureStack(err)
		}
		rows.Close()

		dirEntrs, err := os.ReadDir(tempDir)
		if err != nil {
			return err
		}
		// There should be only one file due to the way we construct the Snowflake source path.
		// However, GET can download multiple files with a common prefix, so this loop future proofs us a bit.
		for _, de := range dirEntrs {
			if de.Type().IsRegular() {
				bytes, err := ioutil.ReadFile(filepath.Join(tempDir, de.Name()))
				if err != nil {
					return err
				}
				_, err = w.Write(bytes)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
