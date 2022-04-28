package sdata

import (
	"database/sql"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachsql"
)

// Tuple is an alias for []interface{}.
// It is used for passing around rows of data.
// The elements of a tuple will always be pointers so the Tuple can
// be passed to sql.Rows.Scan
type Tuple = []interface{}

// func (t Tuple) String() string {
// 	result := []string{}
// 	for i, x := range t {
// 		result = append(result, fmt.Sprintf("[%d] type: %v value: %v", i, reflect.TypeOf(x), reflect.ValueOf(x)))
// 	}
// 	return "Tuple[" + strings.Join(result, ", ") + "]"
// }

// TupleWriter is the type of Writers for structured data.
type TupleWriter interface {
	WriteTuple(row Tuple) error
	Flush() error
}

// TupleReader is a stream of Tuples
type TupleReader interface {
	// Next attempts to read one Tuple into x.
	// If the next data is the wrong shape for x then an error is returned.
	Next(x Tuple) error
}

// MaterializationResult is returned by MaterializeSQL
type MaterializationResult struct {
	ColumnNames []string
	RowCount    uint64
}

// MaterializeSQL reads all the rows from a *sql.Rows, and writes them to tw.
// It flushes tw and returns a MaterializationResult
func MaterializeSQL(tw TupleWriter, rows *sql.Rows) (*MaterializationResult, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, errors.EnsureStack(err)
	}
	cTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, errors.EnsureStack(err)
	}
	row, err := NewTupleFromColumnTypes(cTypes)
	if err != nil {
		return nil, err
	}

	var count uint64
	for rows.Next() {
		if err := rows.Scan(row...); err != nil {
			return nil, errors.EnsureStack(err)
		}
		if err := tw.WriteTuple(row); err != nil {
			return nil, errors.EnsureStack(err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, errors.EnsureStack(err)
	}
	if err := tw.Flush(); err != nil {
		return nil, errors.EnsureStack(err)
	}

	return &MaterializationResult{
		ColumnNames: colNames,
		RowCount:    count,
	}, nil
}

func NewTupleFromColumnTypes(cTypes []*sql.ColumnType) (Tuple, error) {
	row := make(Tuple, len(cTypes))
	for i, cType := range cTypes {
		var err error
		row[i], err = makeTupleElementFromColumnType(cType)
		if err != nil {
			return nil, err
		}
	}
	return row, nil
}

// Copy copies a tuple from r to w. Row is used to indicate the correct shape of read data.
func Copy(r TupleReader, w TupleWriter, row Tuple) (n int, _ error) {
	for {
		err := r.Next(row)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return n, errors.EnsureStack(err)
		}
		if err := w.WriteTuple(row); err != nil {
			return n, errors.EnsureStack(err)
		}
		n++
	}
	return n, nil
}

func NewTupleFromTableInfo(info *pachsql.TableInfo) (Tuple, error) {
	tuple := make(Tuple, len(info.Columns))
	for i, ci := range info.Columns {
		var err error
		tuple[i], err = makeTupleElementFromInformationSchema(ci)
		if err != nil {
			return nil, err
		}
	}
	return tuple, nil
}

// Convert using Null types explicitly
func makeTupleElementFromColumnType(colType *sql.ColumnType) (interface{}, error) {
	if colType == nil {
		return nil, errors.Errorf("colType cannot be nil")
	}

	nullable, ok := colType.Nullable()
	if !ok {
		nullable = true
	}

	fmt.Println(colType.Name(), colType.DatabaseTypeName(), nullable, colType.ScanType())
	switch colType.ScanType() {
	case reflect.TypeOf(bool(false)):
		if nullable {
			return new(sql.NullBool), nil
		}
		return new(bool), nil
	case reflect.TypeOf(int16(0)), reflect.TypeOf(sql.NullInt16{}):
		if nullable {
			return new(sql.NullInt16), nil
		}
		return new(int16), nil
	case reflect.TypeOf(int32(0)), reflect.TypeOf(sql.NullInt32{}):
		if nullable {
			return new(sql.NullInt32), nil
		}
		return new(int32), nil
	case reflect.TypeOf(int64(0)), reflect.TypeOf(sql.NullInt64{}):
		if nullable {
			return new(sql.NullInt64), nil
		}
		return new(int64), nil
	case reflect.TypeOf(float32(0)):
		if nullable {
			return new(sql.NullFloat64), nil
		}
		return new(float32), nil
	case reflect.TypeOf(float64(0)), reflect.TypeOf(sql.NullFloat64{}):
		if nullable {
			return new(sql.NullFloat64), nil
		}
		return new(float64), nil
	case reflect.TypeOf(string("")):
		if nullable {
			return new(sql.NullString), nil
		}
		return new(string), nil
	case reflect.TypeOf(time.Time{}), reflect.TypeOf(sql.NullTime{}):
		if nullable {
			return new(sql.NullTime), nil
		}
		return new(time.Time), nil
	case reflect.TypeOf(byte(0)):
		if nullable {
			return new(sql.NullByte), nil
		}
		return new(byte), nil
	case reflect.TypeOf(sql.RawBytes{}):
		return new(sql.RawBytes), nil
	default:
		return nil, errors.Errorf("Unrecognized type: %v", colType.ScanType())
	}
}

func makeTupleElementFromInformationSchema(colInfo *pachsql.ColumnInfo) (interface{}, error) {
	if colInfo == nil {
		return nil, errors.Errorf("ColumnInfo cannot be nil")
	}
	switch colInfo.DataType {
	case "NUMBER":
		// TODO deal with precision as well
		if colInfo.IsNullable {
			if colInfo.Scale == 0 {
				return new(sql.NullInt64), nil
			} else {
				return new(sql.NullFloat64), nil
			}
		} else {
			if colInfo.Scale == 0 {
				return new(int64), nil
			} else {
				return new(float64), nil
			}
		}
	case "BOOL", "BOOLEAN":
		if colInfo.IsNullable {
			return new(sql.NullBool), nil
		} else {
			return new(bool), nil
		}
	case "SMALLINT", "INT2":
		if colInfo.IsNullable {
			return new(sql.NullInt16), nil
		} else {
			return new(int16), nil
		}
	case "INTEGER", "INT", "INT4":
		if colInfo.IsNullable {
			return new(sql.NullInt32), nil
		} else {
			return new(int32), nil
		}
	case "BIGINT", "INT8":
		if colInfo.IsNullable {
			return new(sql.NullInt64), nil
		} else {
			return new(int64), nil
		}
	case "FLOAT", "FLOAT8", "REAL", "DOUBLE PRECISION":
		if colInfo.IsNullable {
			return new(sql.NullFloat64), nil
		} else {
			return new(float64), nil
		}
	case "VARCHAR", "TEXT", "CHARACTER VARYING":
		if colInfo.IsNullable {
			return new(sql.NullString), nil
		} else {
			return new(string), nil
		}
	case "DATE", "TIMESTAMP", "TIMESTAMP_NTZ", "TIMESTAMP WITHOUT TIME ZONE":
		if colInfo.IsNullable {
			return new(sql.NullTime), nil
		} else {
			return new(time.Time), nil
		}
	default:
		return nil, errors.Errorf("unrecognized type: %v", colInfo.DataType)
	}
}
