package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateBusinessTerm 创建业务术语
func (s *PostgresStore) CreateBusinessTerm(ctx context.Context, term *BusinessTermCreate) (string, error) {
	query := `
		INSERT INTO business_terms (id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, term.Name, term.Description, term.Category,
		term.StandardCode, term.Domain, term.DataTypeStandard, term.ValidationRule, term.Owner, term.Status,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create business term: %w", err)
	}
	return id, nil
}

// GetBusinessTerm 获取业务术语
func (s *PostgresStore) GetBusinessTerm(ctx context.Context, id string) (*BusinessTermRow, error) {
	query := `
		SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
		FROM business_terms
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)

	var term BusinessTermRow
	err := row.Scan(
		&term.ID, &term.Name, &term.Description, &term.Category,
		&term.StandardCode, &term.Domain, &term.DataTypeStandard, &term.ValidationRule, &term.Owner, &term.Status,
		&term.CreatedAt, &term.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("business term not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get business term: %w", err)
	}
	return &term, nil
}

// ListBusinessTerms 列出业务术语
func (s *PostgresStore) ListBusinessTerms(ctx context.Context, category string) ([]*BusinessTermRow, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `
			SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
			FROM business_terms
			WHERE category = $1
			ORDER BY name
		`
		args = append(args, category)
	} else {
		query = `
			SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
			FROM business_terms
			ORDER BY name
		`
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list business terms: %w", err)
	}
	defer rows.Close()

	var terms []*BusinessTermRow
	for rows.Next() {
		var term BusinessTermRow
		err := rows.Scan(
			&term.ID, &term.Name, &term.Description, &term.Category,
			&term.StandardCode, &term.Domain, &term.DataTypeStandard, &term.ValidationRule, &term.Owner, &term.Status,
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
func (s *PostgresStore) UpdateBusinessTerm(ctx context.Context, id string, updates *BusinessTermUpdate) error {
	query := `
		UPDATE business_terms
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    standard_code = COALESCE($4, standard_code),
		    domain = COALESCE($5, domain),
		    data_type_standard = COALESCE($6, data_type_standard),
		    validation_rule = COALESCE($7, validation_rule),
		    owner = COALESCE($8, owner),
		    status = COALESCE($9, status),
		    updated_at = NOW()
		WHERE id = $10
	`
	result, err := s.pool.Exec(ctx, query,
		updates.Name, updates.Description, updates.Category,
		updates.StandardCode, updates.Domain, updates.DataTypeStandard, updates.ValidationRule, updates.Owner, updates.Status,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update business term: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

// DeleteBusinessTerm 删除业务术语
func (s *PostgresStore) DeleteBusinessTerm(ctx context.Context, id string) error {
	// 先清除字段关联
	_, err := s.pool.Exec(ctx, `UPDATE columns SET term_id = NULL WHERE term_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to clear term associations: %w", err)
	}

	result, err := s.pool.Exec(ctx, `DELETE FROM business_terms WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete business term: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

// AssignTermToColumn 分配术语到字段
func (s *PostgresStore) AssignTermToColumn(ctx context.Context, columnID string, termID *string) error {
	query := `UPDATE columns SET term_id = $1, updated_at = NOW() WHERE id = $2`
	result, err := s.pool.Exec(ctx, query, termID, columnID)
	if err != nil {
		return fmt.Errorf("failed to assign term to column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", columnID)
	}
	return nil
}

// GetObjectWithColumns 获取对象及其字段（用于DDL生成）
func (s *PostgresStore) GetObjectWithColumns(ctx context.Context, objectID string) (*SchemaObjectRow, []*ColumnRow, error) {
	obj, err := s.GetSchemaObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	cols, err := s.ListColumnsByObject(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}

	return obj, cols, nil
}

// Transaction implementations for PostgresTxStore

func (t *PostgresTxStore) CreateBusinessTerm(ctx context.Context, term *BusinessTermCreate) (string, error) {
	query := `
		INSERT INTO business_terms (id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, term.Name, term.Description, term.Category,
		term.StandardCode, term.Domain, term.DataTypeStandard, term.ValidationRule, term.Owner, term.Status,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create business term: %w", err)
	}
	return id, nil
}

func (t *PostgresTxStore) GetBusinessTerm(ctx context.Context, id string) (*BusinessTermRow, error) {
	query := `
		SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
		FROM business_terms
		WHERE id = $1
	`
	row := t.tx.QueryRow(ctx, query, id)

	var term BusinessTermRow
	err := row.Scan(
		&term.ID, &term.Name, &term.Description, &term.Category,
		&term.StandardCode, &term.Domain, &term.DataTypeStandard, &term.ValidationRule, &term.Owner, &term.Status,
		&term.CreatedAt, &term.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("business term not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get business term: %w", err)
	}
	return &term, nil
}

func (t *PostgresTxStore) ListBusinessTerms(ctx context.Context, category string) ([]*BusinessTermRow, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `
			SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
			FROM business_terms
			WHERE category = $1
			ORDER BY name
		`
		args = append(args, category)
	} else {
		query = `
			SELECT id, name, description, category, standard_code, domain, data_type_standard, validation_rule, owner, COALESCE(status, 'active') AS status, created_at::text, updated_at::text
			FROM business_terms
			ORDER BY name
		`
	}

	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list business terms: %w", err)
	}
	defer rows.Close()

	var terms []*BusinessTermRow
	for rows.Next() {
		var term BusinessTermRow
		err := rows.Scan(
			&term.ID, &term.Name, &term.Description, &term.Category,
			&term.StandardCode, &term.Domain, &term.DataTypeStandard, &term.ValidationRule, &term.Owner, &term.Status,
			&term.CreatedAt, &term.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan business term: %w", err)
		}
		terms = append(terms, &term)
	}
	return terms, rows.Err()
}

func (t *PostgresTxStore) UpdateBusinessTerm(ctx context.Context, id string, updates *BusinessTermUpdate) error {
	query := `
		UPDATE business_terms
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    standard_code = COALESCE($4, standard_code),
		    domain = COALESCE($5, domain),
		    data_type_standard = COALESCE($6, data_type_standard),
		    validation_rule = COALESCE($7, validation_rule),
		    owner = COALESCE($8, owner),
		    status = COALESCE($9, status),
		    updated_at = NOW()
		WHERE id = $10
	`
	result, err := t.tx.Exec(ctx, query,
		updates.Name, updates.Description, updates.Category,
		updates.StandardCode, updates.Domain, updates.DataTypeStandard, updates.ValidationRule, updates.Owner, updates.Status,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update business term: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

func (t *PostgresTxStore) DeleteBusinessTerm(ctx context.Context, id string) error {
	_, err := t.tx.Exec(ctx, `UPDATE columns SET term_id = NULL WHERE term_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to clear term associations: %w", err)
	}

	result, err := t.tx.Exec(ctx, `DELETE FROM business_terms WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete business term: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("business term not found: %s", id)
	}
	return nil
}

func (t *PostgresTxStore) AssignTermToColumn(ctx context.Context, columnID string, termID *string) error {
	query := `UPDATE columns SET term_id = $1, updated_at = NOW() WHERE id = $2`
	result, err := t.tx.Exec(ctx, query, termID, columnID)
	if err != nil {
		return fmt.Errorf("failed to assign term to column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", columnID)
	}
	return nil
}

func (t *PostgresTxStore) GetObjectWithColumns(ctx context.Context, objectID string) (*SchemaObjectRow, []*ColumnRow, error) {
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
