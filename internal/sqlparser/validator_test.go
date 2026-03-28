package sqlparser

import (
	"testing"
)

func TestValidator_ValidateSelectSQL(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
		errCode string
	}{
		{
			name:    "valid simple select",
			sql:     "SELECT * FROM users WHERE id = 1",
			wantErr: false,
		},
		{
			name:    "valid select with columns",
			sql:     "SELECT id, name, email FROM customers",
			wantErr: false,
		},
		{
			name:    "valid select with where clause",
			sql:     "SELECT * FROM orders WHERE status = 'pending' AND amount > 100",
			wantErr: false,
		},
		{
			name:    "invalid - insert statement",
			sql:     "INSERT INTO users (name) VALUES ('test')",
			wantErr: true,
			errCode: "NOT_SELECT",
		},
		{
			name:    "invalid - update statement",
			sql:     "UPDATE users SET name = 'test' WHERE id = 1",
			wantErr: true,
			errCode: "NOT_SELECT",
		},
		{
			name:    "invalid - delete statement",
			sql:     "DELETE FROM users WHERE id = 1",
			wantErr: true,
			errCode: "NOT_SELECT",
		},
		{
			name:    "invalid - drop statement",
			sql:     "DROP TABLE users",
			wantErr: true,
			errCode: "NOT_SELECT",
		},
		{
			name:    "invalid - dangerous function xp_cmdshell",
			sql:     "SELECT * FROM users WHERE name = xp_cmdshell('dir')",
			wantErr: true,
			errCode: "DANGEROUS_FUNCTION",
		},
		{
			name:    "invalid - dangerous function benchmark",
			sql:     "SELECT BENCHMARK(1000000, MD5('test'))",
			wantErr: true,
			errCode: "DANGEROUS_FUNCTION",
		},
		{
			name:    "invalid - dangerous keyword semicolon",
			sql:     "SELECT * FROM users; DROP TABLE users",
			wantErr: true,
			errCode: "DANGEROUS_KEYWORD",
		},
		{
			name:    "invalid - union",
			sql:     "SELECT * FROM users UNION SELECT * FROM admins",
			wantErr: true,
			errCode: "UNION_NOT_ALLOWED",
		},
		{
			name:    "valid - with join",
			sql:     "SELECT u.*, o.amount FROM users u JOIN orders o ON u.id = o.user_id",
			wantErr: false,
		},
		{
			name:    "valid - with group by",
			sql:     "SELECT status, COUNT(*) FROM orders GROUP BY status",
			wantErr: false,
		},
		{
			name:    "valid - with order by",
			sql:     "SELECT * FROM users ORDER BY created_at DESC",
			wantErr: false,
		},
		{
			name:    "valid - with limit",
			sql:     "SELECT * FROM users LIMIT 100",
			wantErr: false,
		},
		{
			name:    "valid - subquery",
			sql:     "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSelectSQL(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSelectSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errCode != "" {
				if valErr, ok := err.(*ValidationError); ok {
					if valErr.Code != tt.errCode {
						t.Errorf("ValidateSelectSQL() error code = %v, want %v", valErr.Code, tt.errCode)
					}
				} else {
					t.Errorf("ValidateSelectSQL() error is not ValidationError: %v", err)
				}
			}
		})
	}
}

func TestValidator_SanitizeAndAddLimit(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		sql       string
		limit     int
		wantLimit string
		wantErr   bool
	}{
		{
			name:      "add limit to simple select",
			sql:       "SELECT * FROM users",
			limit:     100,
			wantLimit: "select * from users limit 100",
			wantErr:   false,
		},
		{
			name:      "keep existing limit",
			sql:       "SELECT * FROM users LIMIT 50",
			limit:     100,
			wantLimit: "select * from users limit 50",
			wantErr:   false,
		},
		{
			name:      "add limit to complex select",
			sql:       "SELECT u.*, o.amount FROM users u JOIN orders o ON u.id = o.user_id WHERE u.status = 'active'",
			limit:     1000,
			wantLimit: "select u.*, o.amount from users as u join orders as o on u.id = o.user_id where u.`status` = 'active' limit 1000",
			wantErr:   false,
		},
		{
			name:    "invalid sql",
			sql:     "SELECT * FROM",
			limit:   100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.SanitizeAndAddLimit(tt.sql, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeAndAddLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantLimit {
				t.Errorf("SanitizeAndAddLimit() = %v, want %v", got, tt.wantLimit)
			}
		})
	}
}

func TestValidator_ExtractTableNames(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name       string
		sql        string
		wantTables []string
		wantErr    bool
	}{
		{
			name:       "single table",
			sql:        "SELECT * FROM users",
			wantTables: []string{"users"},
			wantErr:    false,
		},
		{
			name:       "table with schema",
			sql:        "SELECT * FROM public.users",
			wantTables: []string{"public.users"},
			wantErr:    false,
		},
		{
			name:       "join tables",
			sql:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			wantTables: []string{"users", "orders"},
			wantErr:    false,
		},
		{
			name:       "subquery",
			sql:        "SELECT * FROM (SELECT * FROM users) AS u",
			wantTables: []string{"users"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.ExtractTableNames(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTableNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.wantTables) {
				t.Errorf("ExtractTableNames() = %v, want %v", got, tt.wantTables)
			}
		})
	}
}
