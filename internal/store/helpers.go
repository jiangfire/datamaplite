package store

import (
	"database/sql"

	"go.uber.org/zap"
)

func ignoreError(err error) {}

func closeSQLRows(rows *sql.Rows) {
	if rows == nil {
		return
	}
	ignoreError(rows.Close())
}

func rollbackSQLTx(tx *sql.Tx, logger *zap.Logger) {
	if tx == nil {
		return
	}
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone && logger != nil {
		logger.Warn("failed to rollback sqlite transaction", zap.Error(err))
	}
}
