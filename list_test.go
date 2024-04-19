package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequestedList(t *testing.T) {
	t.Parallel()

	columns, err := parseRequestedColumns("ID,NAME")
	assert.Equal(t, []string{"ID", "NAME"}, columns)
	require.NoError(t, err)

	columns, err = parseRequestedColumns(" ID, NAME")
	assert.Equal(t, []string{"ID", "NAME"}, columns)
	require.NoError(t, err)

	defaultColumns, err := parseRequestedColumns(defaultListColumns)
	require.NoError(t, err)

	columns, err = parseRequestedColumns("")
	assert.Equal(t, defaultColumns, columns)
	require.NoError(t, err)

	columns, err = parseRequestedColumns("ID,NAME,BAD")
	assert.Equal(t, []string(nil), columns)
	require.Error(t, err)
}
