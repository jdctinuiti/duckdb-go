package duckdb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/duckdb/duckdb-go/v2/mapping"
)

func TestWithNativeConnection(t *testing.T) {
	connector, err := NewConnector(":memory:", nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connector.Close()) })
	owned, err := connector.Connect(context.Background())
	require.NoError(t, err)
	conn := owned.(*Conn)

	want := errors.New("callback error")
	err = WithNativeConnection(conn, func(handle mapping.Connection) error {
		require.Equal(t, conn.id, extractConnId(handle))
		return want
	})
	require.Same(t, want, err)
	require.Error(t, WithNativeConnection(conn, nil))
	require.NoError(t, conn.Close())
	invalid := func(mapping.Connection) error {
		t.Fatal("invalid connection reached callback")
		return nil
	}
	require.ErrorIs(t, WithNativeConnection(conn, invalid), errClosedCon)
	require.ErrorIs(t, WithNativeConnection(nil, invalid), errInvalidCon)
}

func TestGetTableNames(t *testing.T) {
	db := openDbWrapper(t, ``)
	defer closeDbWrapper(t, db)

	conn := openConnWrapper(t, db, context.Background())
	defer closeConnWrapper(t, conn)

	tests := []struct {
		name           string
		query          string
		qualified      bool
		expectedTables []string
		expectedError  string
	}{
		{
			name:           "valid query with multiple tables, qualified",
			query:          `SELECT * FROM schema1.table1, catalog3."schema.2"."table.2"`,
			qualified:      true,
			expectedTables: []string{"schema1.table1", `catalog3."schema.2"."table.2"`},
		},
		{
			name:           "valid query with multiple tables, unqualified",
			query:          `SELECT * FROM schema1.table1, catalog3."schema.2"."table.2"`,
			qualified:      false,
			expectedTables: []string{"table1", "table.2"},
		},
		{
			name:           "valid query with no tables",
			query:          "SELECT 1 as num",
			expectedTables: nil,
		},
		{
			name:          "invalid query syntax",
			query:         "SELECT * FROM WHERE",
			expectedError: "Parser Error: syntax error at or near \"WHERE\"",
		},
		{
			name:          "empty query",
			query:         "",
			expectedError: "empty query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableNames, err := GetTableNames(conn, tt.query, tt.qualified)
			if tt.expectedError != "" {
				require.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, tableNames)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedTables, tableNames)
			}
		})
	}
}
