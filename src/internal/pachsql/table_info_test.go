package pachsql_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/pachyderm/pachyderm/v2/src/internal/dockertestenv"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachsql"
	"github.com/pachyderm/pachyderm/v2/src/internal/require"
	"github.com/pachyderm/pachyderm/v2/src/internal/testsnowflake"
)

func TestGetTableInfo(t *testing.T) {
	type testCase struct {
		Name     string
		NewDB    func(t testing.TB) *pachsql.DB
		NewTable func(*pachsql.DB) error
		Expected *pachsql.TableInfo
	}
	tcs := []testCase{
		{
			Name:  "Postgres",
			NewDB: dockertestenv.NewPostgres,
			NewTable: func(db *pachsql.DB) error {
				return pachsql.CreateTestTable(db, "test_table", pachsql.TestRow{})
			},
			Expected: &pachsql.TableInfo{
				"test_table",
				"public",
				[]*pachsql.ColumnInfo{
					{"c_id", "INTEGER", false, 0, 0},
					{"c_smallint", "SMALLINT", false, 0, 0},
					{"c_int", "INTEGER", false, 0, 0},
					{"c_bigint", "BIGINT", false, 0, 0},
					{"c_float", "DOUBLE PRECISION", false, 0, 0},
					{"c_varchar", "CHARACTER VARYING", false, 0, 0},
					{"c_time", "TIMESTAMP WITHOUT TIME ZONE", false, 0, 0},
					{"c_smallint_null", "SMALLINT", true, 0, 0},
					{"c_int_null", "INTEGER", true, 0, 0},
					{"c_bigint_null", "BIGINT", true, 0, 0},
					{"c_float_null", "DOUBLE PRECISION", true, 0, 0},
					{"c_varchar_null", "CHARACTER VARYING", true, 0, 0},
					{"c_time_null", "TIMESTAMP WITHOUT TIME ZONE", true, 0, 0},
				},
			},
		},
		{
			Name:  "MySQL",
			NewDB: dockertestenv.NewMySQL,
			NewTable: func(db *pachsql.DB) error {
				return pachsql.CreateTestTable(db, "public.test_table", pachsql.TestRow{})
			},
			Expected: &pachsql.TableInfo{
				"test_table",
				"public",
				[]*pachsql.ColumnInfo{
					{"c_id", "INT", false, 0, 0},
					{"c_smallint", "SMALLINT", false, 0, 0},
					{"c_int", "INT", false, 0, 0},
					{"c_bigint", "BIGINT", false, 0, 0},
					{"c_float", "FLOAT", false, 0, 0},
					{"c_varchar", "VARCHAR", false, 0, 0},
					{"c_time", "TIMESTAMP", false, 0, 0},
					{"c_smallint_null", "SMALLINT", true, 0, 0},
					{"c_int_null", "INT", true, 0, 0},
					{"c_bigint_null", "BIGINT", true, 0, 0},
					{"c_float_null", "FLOAT", true, 0, 0},
					{"c_varchar_null", "VARCHAR", true, 0, 0},
					{"c_time_null", "TIMESTAMP", true, 0, 0},
				},
			},
		},
		{
			Name:  "Snowflake",
			NewDB: testsnowflake.NewSnowSQL,
			NewTable: func(db *pachsql.DB) error {
				return pachsql.CreateTestTable(db, "test_table", pachsql.TestRow{})
			},
			Expected: &pachsql.TableInfo{
				"test_table",
				"public",
				[]*pachsql.ColumnInfo{
					{"C_ID", "NUMBER", false, 0, 0},
					{"C_SMALLINT", "NUMBER", false, 0, 0},
					{"C_INT", "NUMBER", false, 0, 0},
					{"C_BIGINT", "NUMBER", false, 0, 0},
					{"C_FLOAT", "FLOAT", false, 0, 0},
					{"C_VARCHAR", "TEXT", false, 0, 0},
					{"C_TIME", "TIMESTAMP_NTZ", false, 0, 0},
					{"C_SMALLINT_NULL", "NUMBER", true, 0, 0},
					{"C_INT_NULL", "NUMBER", true, 0, 0},
					{"C_BIGINT_NULL", "NUMBER", true, 0, 0},
					{"C_FLOAT_NULL", "FLOAT", true, 0, 0},
					{"C_VARCHAR_NULL", "TEXT", true, 0, 0},
					{"C_TIME_NULL", "TIMESTAMP_NTZ", true, 0, 0},
				},
			},
		},
	}
	ctx := context.Background()
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			db := tc.NewDB(t)
			// For mysql, we created a database named public via NewMySQL
			require.NoError(t, tc.NewTable(db))
			info, err := pachsql.GetTableInfo(ctx, db, "test_table")
			require.NoError(t, err)
			require.Len(t, info.Columns, reflect.TypeOf(pachsql.TestRow{}).NumField())
			require.Equal(t, tc.Expected, info)
		})
	}
}
