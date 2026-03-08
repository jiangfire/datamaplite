package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateBusinessTerm 创建业务术语
func (s *SQLiteStore) CreateBusinessTerm(ctx context.Context, term *BusinessTermCreate) (string, error) {
	query := `
		INSERT INTO business_terms (id, name, description, category)
		VALUES (?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, term.Name, term.Description, term.Category,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create business term: %w", err)
	}
	return id, nil
}

// GetBusinessTerm 获取业务术语
func (s *SQLiteStore) GetBusinessTerm(ctx context.Context, id string) (*BusinessTermRow, error) {
	query := `
		SELECT id, name, description, category, created_at, updated_at
		FROM business_terms
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var term BusinessTermRow
	err := row.Scan(
		&term.ID, &term.Name, &term.Description, &term.Category,
		&term.CreatedAt, &term.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("business term not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get business term: %w", err)
	}
	return &term, nil
}

// ListBusinessTerms 列出业务术语
func (s *SQLiteStore) ListBusinessTerms(ctx context.Context, category string) ([]*BusinessTermRow, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `
			SELECT id, name, description, category, created_at, updated_at
			FROM business_terms
			WHERE category = ?
			ORDER BY name
		`
		args = append(args, category)
	} else {
		query = `
			SELECT id, name, description, category, created_at, updated_at
			FROM business_terms
			ORDER BY name
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list business terms: %w", err)
	}
	defer rows.Close()

	var terms []*BusinessTermRow
	for rows.Next() {
		var term BusinessTermRow
		err := rows.Scan(
			&term.ID, &term.Name, &term.Description, &term.Category,
			&term.CreatedAt, &term.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan business term: %w", err)
		}
		terms = append(terms, &term)
	}
	return terms, rows.Err()
}

// UpdateBusinessTerm 更新业务术语
func (s *SQLiteStore) UpdateBusinessTerm(ctx context.Context, id string, updates *BusinessTermUpdate) error {
	query := `
		UPDATE business_terms
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    updated_at = datetime('now')
		WHERE id = $4
	`
	result, err := s.db.ExecContext(ctx, query,
		updates.Name, updates.Description, updates.Category, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update business term: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

// DeleteBusinessTerm 删除业务术语
func (s *SQLiteStore) DeleteBusinessTerm(ctx context.Context, id string) error {
	// 先清除字段关联
	_, err := s.db.ExecContext(ctx, `UPDATE columns SET term_id = NULL WHERE term_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to clear term associations: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM business_terms WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete business term: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

// AssignTermToColumn 分配术语到字段
func (s *SQLiteStore) AssignTermToColumn(ctx context.Context, columnID string, termID *string) error {
	query := `UPDATE columns SET term_id = ?, updated_at = datetime('now') WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, termID, columnID)
	if err != nil {
		return fmt.Errorf("failed to assign term to column: %w", err)
	}
	return nil
}

// GetObjectWithColumns 获取对象及其字段（用于DDL生成）
func (s *SQLiteStore) GetObjectWithColumns(ctx context.Context, objectID string) (*SchemaObjectRow, []*ColumnRow, error) {
	// 获取对象
	obj, err := s.GetSchemaObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	// 获取字段
	cols, err := s.ListColumnsByObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	return obj, cols, nil
}

// Transaction implementations for SQLiteTxStore

func (t *SQLiteTxStore) CreateBusinessTerm(ctx context.Context, term *BusinessTermCreate) (string, error) {
	query := `
		INSERT INTO business_terms (id, name, description, category)
		VALUES (?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, term.Name, term.Description, term.Category,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create business term: %w", err)
	}
	return id, nil
}

func (t *SQLiteTxStore) GetBusinessTerm(ctx context.Context, id string) (*BusinessTermRow, error) {
	query := `
		SELECT id, name, description, category, created_at, updated_at
		FROM business_terms
		WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)

	var term BusinessTermRow
	err := row.Scan(
		&term.ID, &term.Name, &term.Description, &term.Category,
		&term.CreatedAt, &term.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("business term not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get business term: %w", err)
	}
	return &term, nil
}

func (t *SQLiteTxStore) ListBusinessTerms(ctx context.Context, category string) ([]*BusinessTermRow, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `
			SELECT id, name, description, category, created_at, updated_at
			FROM business_terms
			WHERE category = ?
			ORDER BY name
		`
		args = append(args, category)
	} else {
		query = `
			SELECT id, name, description, category, created_at, updated_at
			FROM business_terms
			ORDER BY name
		`
	}

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list business terms: %w", err)
	}
	defer rows.Close()

	var terms []*BusinessTermRow
	for rows.Next() {
		var term BusinessTermRow
		err := rows.Scan(
			&term.ID, &term.Name, &term.Description, &term.Category,
			&term.CreatedAt, &term.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan business term: %w", err)
		}
		terms = append(terms, &term)
	}
	return terms, rows.Err()
}

func (t *SQLiteTxStore) UpdateBusinessTerm(ctx context.Context, id string, updates *BusinessTermUpdate) error {
	query := `
		UPDATE business_terms
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    updated_at = datetime('now')
		WHERE id = $4
	`
	result, err := t.tx.ExecContext(ctx, query,
		updates.Name, updates.Description, updates.Category, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update business term: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) DeleteBusinessTerm(ctx context.Context, id string) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE columns SET term_id = NULL WHERE term_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to clear term associations: %w", err)
	}

	result, err := t.tx.ExecContext(ctx, `DELETE FROM business_terms WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete business term: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) AssignTermToColumn(ctx context.Context, columnID string, termID *string) error {
	query := `UPDATE columns SET term_id = ?, updated_at = datetime('now') WHERE id = ?`
	_, err := t.tx.ExecContext(ctx, query, termID, columnID)
	if err != nil {
		return fmt.Errorf("failed to assign term to column: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) GetObjectWithColumns(ctx context.Context, objectID string) (*SchemaObjectRow, []*ColumnRow, error) {
	obj, err := t.GetSchemaObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	cols, err := t.ListColumnsByObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	return obj, cols, nil
}
