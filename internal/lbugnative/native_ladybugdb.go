//go:build ladybugdb

package lbugnative

/*
#include "lbug.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"path/filepath"
	"strconv"
	"unsafe"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
)

type nativeDatabase struct {
	db     C.lbug_database
	closed bool
}

func openNativeDatabase(path string, readOnly bool) (*nativeDatabase, error) {
	cPath := C.CString(filepath.ToSlash(path))
	defer C.free(unsafe.Pointer(cPath))

	config := C.lbug_default_system_config()
	config.read_only = C.bool(readOnly)

	var db C.lbug_database
	if state := C.lbug_database_init(cPath, config, &db); state != C.LbugSuccess {
		return nil, fmt.Errorf("lbug_database_init failed with state %d", int(state))
	}
	return &nativeDatabase{db: db}, nil
}

func (db *nativeDatabase) Close() {
	if db == nil || db.closed {
		return
	}
	C.lbug_database_destroy(&db.db)
	db.closed = true
}

func (db *nativeDatabase) OpenConnection() (*nativeConnection, error) {
	var conn C.lbug_connection
	if state := C.lbug_connection_init(&db.db, &conn); state != C.LbugSuccess {
		return nil, fmt.Errorf("lbug_connection_init failed with state %d", int(state))
	}
	return &nativeConnection{conn: conn}, nil
}

type nativeConnection struct {
	conn   C.lbug_connection
	closed bool
}

func (conn *nativeConnection) Close() {
	if conn == nil || conn.closed {
		return
	}
	C.lbug_connection_destroy(&conn.conn)
	conn.closed = true
}

func (conn *nativeConnection) Query(query string) (*nativeResult, error) {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	var result C.lbug_query_result
	state := C.lbug_connection_query(&conn.conn, cQuery, &result)
	if state != C.LbugSuccess || !bool(C.lbug_query_result_is_success(&result)) {
		err := queryResultError(&result, fmt.Sprintf("query failed with state %d", int(state)))
		C.lbug_query_result_destroy(&result)
		return nil, err
	}
	return &nativeResult{result: result}, nil
}

type nativeResult struct {
	result C.lbug_query_result
	closed bool
}

func (result *nativeResult) Close() {
	if result == nil || result.closed {
		return
	}
	C.lbug_query_result_destroy(&result.result)
	result.closed = true
}

func (result *nativeResult) Rows() ([]lbugruntime.Row, error) {
	columns, err := result.columns()
	if err != nil {
		return nil, err
	}

	var rows []lbugruntime.Row
	for bool(C.lbug_query_result_has_next(&result.result)) {
		var tuple C.lbug_flat_tuple
		if state := C.lbug_query_result_get_next(&result.result, &tuple); state != C.LbugSuccess {
			return nil, fmt.Errorf("lbug_query_result_get_next failed with state %d", int(state))
		}

		row, err := tupleRow(&tuple, columns)
		C.lbug_flat_tuple_destroy(&tuple)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (result *nativeResult) columns() ([]string, error) {
	count := int(C.lbug_query_result_get_num_columns(&result.result))
	columns := make([]string, 0, count)
	for index := 0; index < count; index++ {
		var cName *C.char
		if state := C.lbug_query_result_get_column_name(&result.result, C.uint64_t(index), &cName); state != C.LbugSuccess {
			return nil, fmt.Errorf("lbug_query_result_get_column_name(%d) failed with state %d", index, int(state))
		}
		columns = append(columns, C.GoString(cName))
		C.lbug_destroy_string(cName)
	}
	return columns, nil
}

func tupleRow(tuple *C.lbug_flat_tuple, columns []string) (lbugruntime.Row, error) {
	row := make(lbugruntime.Row, len(columns))
	for index, column := range columns {
		value, err := tupleString(tuple, index)
		if err != nil {
			return nil, err
		}
		row[column] = value
	}
	return row, nil
}

func tupleString(tuple *C.lbug_flat_tuple, index int) (string, error) {
	var value C.lbug_value
	if state := C.lbug_flat_tuple_get_value(tuple, C.uint64_t(index), &value); state != C.LbugSuccess {
		return "", fmt.Errorf("lbug_flat_tuple_get_value(%d) failed with state %d", index, int(state))
	}
	defer C.lbug_value_destroy(&value)

	var cString *C.char
	if state := C.lbug_value_get_string(&value, &cString); state == C.LbugSuccess {
		defer C.lbug_destroy_string(cString)
		return C.GoString(cString), nil
	}

	rendered := C.lbug_value_to_string(&value)
	if rendered == nil {
		return "", nil
	}
	defer C.lbug_destroy_string(rendered)
	if unquoted, err := strconv.Unquote(C.GoString(rendered)); err == nil {
		return unquoted, nil
	}
	return C.GoString(rendered), nil
}

func queryResultError(result *C.lbug_query_result, fallback string) error {
	cMessage := C.lbug_query_result_get_error_message(result)
	if cMessage == nil {
		return fmt.Errorf("%s", fallback)
	}
	defer C.lbug_destroy_string(cMessage)
	message := C.GoString(cMessage)
	if message == "" {
		return fmt.Errorf("%s", fallback)
	}
	return fmt.Errorf("%s: %s", fallback, message)
}
