package transforms

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pachyderm/pachyderm/v2/src/internal/randutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/testsnowflake"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSnowflakeBulkExport(t *testing.T) {
	ctx := context.Background()

	// COPY INTO <location> + LIST <location>
	inputDir, outputDir := t.TempDir(), t.TempDir()
	writeCronFile(t, inputDir)

	db := testsnowflake.NewSnowSQL(t) // this creates an ephemeral database, but we won't use it
	defer db.Close()

	nRows := 10
	tableName := createTableWithData(t, db, nRows)

	err := SnowflakeStageFiles(ctx, db, SnowflakeStageFilesParams{
		Logger: logrus.StandardLogger(),

		InputDir:  inputDir,
		OutputDir: outputDir,

		Query:         fmt.Sprintf("select * from %s", tableName),
		InternalStage: fmt.Sprintf("%%%s", tableName), // use Snowflake Table Stage
		FileFormat:    "CSV",
		PartitionBy:   "id",
		MaxFileSize:   16777216,
	})
	require.NoError(t, err)

	dirEntrs, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Len(t, dirEntrs, nRows)

	// GET <location>

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
