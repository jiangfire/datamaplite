package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateColumnMapping 创建字段映射
func (s *PostgresStore) CreateColumnMapping(ctx context.Context, mapping *ColumnMappingCreate) error {
	query := `
		INSERT INTO column_mappings (id, source_column_id, target_column_id, mapping_type, confidence)
		VALUES ($1, $2, $3, $4, $5)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, mapping.SourceColumnID, mapping.TargetColumnID, mapping.MappingType, mapping.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to create column mapping: %w", err)
	}
	return nil
}

// GetColumnMappings 获取字段的映射关系
func (s *PostgresStore) GetColumnMappings(ctx context.Context, columnID string) ([]*ColumnMappingRow, error) {
	query := `
		SELECT m.id, m.source_column_id, m.target_column_id, m.mapping_type, m.confidence, m.created_at::text,
		       c.name as target_column_name, o.name as target_object_name, ds.name as target_source_name
		FROM column_mappings m
		JOIN columns c ON m.target_column_id = c.id
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE m.source_column_id = $1
	`
	rows, err := s.pool.Query(ctx, query, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column mappings: %w", err)
	}
	defer rows.Close()

	var mappings []*ColumnMappingRow
	for rows.Next() {
		var m ColumnMappingRow
		err := rows.Scan(
			&m.ID, &m.SourceColumnID, &m.TargetColumnID, &m.MappingType, &m.Confidence, &m.CreatedAt,
			&m.TargetColumnName, &m.TargetObjectName, &m.TargetSourceName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}
		mappings = append(mappings, &m)
	}
	return mappings, rows.Err()
}

// DeleteColumnMapping 删除字段映射
func (s *PostgresStore) DeleteColumnMapping(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM column_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column mapping: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column mapping not found: %s", id)
	}
	return nil
}

// CreateLineageEdge 创建血缘边
func (s *PostgresStore) CreateLineageEdge(ctx context.Context, edge *LineageEdgeCreate) error {
	query := `
		INSERT INTO lineage_edges (id, source_id, target_id, source_type, target_type, transform_sql, job_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, edge.SourceID, edge.TargetID, edge.SourceType, edge.TargetType, edge.TransformSQL, edge.JobName,
	)
	if err != nil {
		return fmt.Errorf("failed to create lineage edge: %w", err)
	}
	return nil
}

// GetLineageUpward 获取上游血缘
func (s *PostgresStore) GetLineageUpward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error) {
	if depth <= 0 {
		depth = 10
	}

	query := `
		WITH RECURSIVE lineage AS (
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       1 as depth
			FROM lineage_edges e
			WHERE e.target_id = $1 AND e.target_type = 'column'
			UNION ALL
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       l.depth + 1
			FROM lineage_edges e
			JOIN lineage l ON e.target_id = l.source_id AND e.target_type = l.source_type
			WHERE l.depth < $2
		)
		SELECT l.id, l.source_id, l.target_id, l.source_type, l.target_type,
		       l.transform_sql, l.job_name, l.created_at
		FROM lineage l
		ORDER BY l.depth, l.created_at
	`

	rows, err := s.pool.Query(ctx, query, columnID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage upward: %w", err)
	}
	defer rows.Close()

	var edges []*LineageEdgeRow
	for rows.Next() {
		var e LineageEdgeRow
		err := rows.Scan(
			&e.ID, &e.SourceID, &e.TargetID, &e.SourceType, &e.TargetType,
			&e.TransformSQL, &e.JobName, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage edge: %w", err)
		}
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}

// GetLineageDownward 获取下游血缘
func (s *PostgresStore) GetLineageDownward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error) {
	if depth <= 0 {
		depth = 10
	}

	query := `
		WITH RECURSIVE lineage AS (
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       1 as depth
			FROM lineage_edges e
			WHERE e.source_id = $1 AND e.source_type = 'column'
			UNION ALL
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       l.depth + 1
			FROM lineage_edges e
			JOIN lineage l ON e.source_id = l.target_id AND e.source_type = l.target_type
			WHERE l.depth < $2
		)
		SELECT l.id, l.source_id, l.target_id, l.source_type, l.target_type,
		       l.transform_sql, l.job_name, l.created_at
		FROM lineage l
		ORDER BY l.depth, l.created_at
	`

	rows, err := s.pool.Query(ctx, query, columnID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage downward: %w", err)
	}
	defer rows.Close()

	var edges []*LineageEdgeRow
	for rows.Next() {
		var e LineageEdgeRow
		err := rows.Scan(
			&e.ID, &e.SourceID, &e.TargetID, &e.SourceType, &e.TargetType,
			&e.TransformSQL, &e.JobName, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage edge: %w", err)
		}
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}

// Transaction implementations for PostgresTxStore

func (t *PostgresTxStore) CreateColumnMapping(ctx context.Context, mapping *ColumnMappingCreate) error {
	query := `
		INSERT INTO column_mappings (id, source_column_id, target_column_id, mapping_type, confidence)
		VALUES ($1, $2, $3, $4, $5)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, mapping.SourceColumnID, mapping.TargetColumnID, mapping.MappingType, mapping.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to create column mapping: %w", err)
	}
	return nil
}

func (t *PostgresTxStore) GetColumnMappings(ctx context.Context, columnID string) ([]*ColumnMappingRow, error) {
	query := `
		SELECT m.id, m.source_column_id, m.target_column_id, m.mapping_type, m.confidence, m.created_at::text,
		       c.name as target_column_name, o.name as target_object_name, ds.name as target_source_name
		FROM column_mappings m
		JOIN columns c ON m.target_column_id = c.id
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE m.source_column_id = $1
	`
	rows, err := t.tx.Query(ctx, query, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column mappings: %w", err)
	}
	defer rows.Close()

	var mappings []*ColumnMappingRow
	for rows.Next() {
		var m ColumnMappingRow
		err := rows.Scan(
			&m.ID, &m.SourceColumnID, &m.TargetColumnID, &m.MappingType, &m.Confidence, &m.CreatedAt,
			&m.TargetColumnName, &m.TargetObjectName, &m.TargetSourceName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}
		mappings = append(mappings, &m)
	}
	return mappings, rows.Err()
}

func (t *PostgresTxStore) DeleteColumnMapping(ctx context.Context, id string) error {
	result, err := t.tx.Exec(ctx, `DELETE FROM column_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column mapping: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column mapping not found: %s", id)
	}
	return nil
}

func (t *PostgresTxStore) CreateLineageEdge(ctx context.Context, edge *LineageEdgeCreate) error {
	query := `
		INSERT INTO lineage_edges (id, source_id, target_id, source_type, target_type, transform_sql, job_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, edge.SourceID, edge.TargetID, edge.SourceType, edge.TargetType, edge.TransformSQL, edge.JobName,
	)
	if err != nil {
		return fmt.Errorf("failed to create lineage edge: %w", err)
	}
	return nil
}

func (t *PostgresTxStore) GetLineageUpward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error) {
	if depth <= 0 {
		depth = 10
	}

	query := `
		WITH RECURSIVE lineage AS (
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       1 as depth
			FROM lineage_edges e
			WHERE e.target_id = $1 AND e.target_type = 'column'
			UNION ALL
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       l.depth + 1
			FROM lineage_edges e
			JOIN lineage l ON e.target_id = l.source_id AND e.target_type = l.source_type
			WHERE l.depth < $2
		)
		SELECT l.id, l.source_id, l.target_id, l.source_type, l.target_type,
		       l.transform_sql, l.job_name, l.created_at
		FROM lineage l
		ORDER BY l.depth, l.created_at
	`

	rows, err := t.tx.Query(ctx, query, columnID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage upward: %w", err)
	}
	defer rows.Close()

	var edges []*LineageEdgeRow
	for rows.Next() {
		var e LineageEdgeRow
		err := rows.Scan(
			&e.ID, &e.SourceID, &e.TargetID, &e.SourceType, &e.TargetType,
			&e.TransformSQL, &e.JobName, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage edge: %w", err)
		}
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}

func (t *PostgresTxStore) GetLineageDownward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error) {
	if depth <= 0 {
		depth = 10
	}

	query := `
		WITH RECURSIVE lineage AS (
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       1 as depth
			FROM lineage_edges e
			WHERE e.source_id = $1 AND e.source_type = 'column'
			UNION ALL
			SELECT e.id, e.source_id, e.target_id, e.source_type, e.target_type,
			       e.transform_sql, e.job_name, e.created_at::text,
			       l.depth + 1
			FROM lineage_edges e
			JOIN lineage l ON e.source_id = l.target_id AND e.source_type = l.target_type
			WHERE l.depth < $2
		)
		SELECT l.id, l.source_id, l.target_id, l.source_type, l.target_type,
		       l.transform_sql, l.job_name, l.created_at
		FROM lineage l
		ORDER BY l.depth, l.created_at
	`

	rows, err := t.tx.Query(ctx, query, columnID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage downward: %w", err)
	}
	defer rows.Close()

	var edges []*LineageEdgeRow
	for rows.Next() {
		var e LineageEdgeRow
		err := rows.Scan(
			&e.ID, &e.SourceID, &e.TargetID, &e.SourceType, &e.TargetType,
			&e.TransformSQL, &e.JobName, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage edge: %w", err)
		}
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}
