package service

import "database/sql"

func ignoreError(err error) {}

func closeSQLRows(rows *sql.Rows) {
	if rows == nil {
		return
	}
	ignoreError(rows.Close())
}
