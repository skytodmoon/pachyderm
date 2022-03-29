package transforms

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pachyderm/pachyderm/v2/src/internal/randutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/testsnowflake"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSnowflakeBulkExport(t *testing.T) {
	// Two parts:
	// 1. Run COPY INTO to stage files
	// 2. Run GET to download those staged files
	// Note: in real Pachyderm systems, these parts would be run as separate pipelines
	ctx := context.Background()
	log := logrus.StandardLogger()
	db := testsnowflake.NewSnowSQL(t)
	defer db.Close()

	// SnowflakeStageFiles
	inputDir, outputDir := t.TempDir(), t.TempDir()

	nRows := 10
	tableName := createTableWithData(t, db, nRows)
	internalStage := fmt.Sprintf("%%%s", tableName) // use Snowflake Table Stage

	err := SnowflakeStageFiles(ctx, db, SnowflakeStageFilesParams{
		Logger: log,

		InputDir:  inputDir,
		OutputDir: outputDir,

		Query:         fmt.Sprintf("select * from %s", tableName),
		InternalStage: internalStage,
		FileFormat:    "CSV",
		Compression:   "NONE",
		PartitionBy:   "id",
		MaxFileSize:   16777216,
	})
	require.NoError(t, err)

	dirEntrs, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Len(t, dirEntrs, nRows)

	// SnowflakeGet
	getOutputDir := t.TempDir()
	SnowflakeGet(ctx, db, SnowflakeGetParams{
		Logger:    log,
		InputDir:  outputDir,
		OutputDir: getOutputDir,
	})

	dirEntrs, err = os.ReadDir(getOutputDir)
	require.NoError(t, err)
	require.Len(t, dirEntrs, nRows)
	for _, d := range dirEntrs {
		data, err := ioutil.ReadFile(filepath.Join(getOutputDir, d.Name()))
		require.NoError(t, err)
		t.Log(d.Name(), string(data))
	}
}

func createTableWithData(t *testing.T, db *sqlx.DB, n int) string {
	tableName := "test_data"
	_, err := db.Exec(fmt.Sprintf(`
	CREATE TABLE %s (
		id INT PRIMARY KEY,
		col_a VARCHAR(100)
	)`, tableName))
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		_, err := db.Exec(`INSERT INTO test_data (id, col_a) VALUES (?, ?)`, i, randutil.UniqueString(""))
		require.NoError(t, err)
	}
	return tableName
}
