package model

import "time"

// DQRuleType 数据质量规则类型
type DQRuleType string

const (
	DQRuleTypeNotNull      DQRuleType = "not_null"
	DQRuleTypeUnique       DQRuleType = "unique"
	DQRuleTypeRegex        DQRuleType = "regex"
	DQRuleTypeRange        DQRuleType = "range"
	DQRuleTypeEnum         DQRuleType = "enum"
	DQRuleTypeCustomSQL    DQRuleType = "custom_sql"
	DQRuleTypeReferential  DQRuleType = "referential"
)

// DQSeverity 数据质量问题严重级别
type DQSeverity string

const (
	DQSeverityError   DQSeverity = "error"
	DQSeverityWarning DQSeverity = "warning"
	DQSeverityInfo    DQSeverity = "info"
)

// DQResultStatus 检测结果状态
type DQResultStatus string

const (
	DQResultStatusPassed DQResultStatus = "passed"
	DQResultStatusFailed DQResultStatus = "failed"
	DQResultStatusError  DQResultStatus = "error"
)

// DQRule 数据质量规则
type DQRule struct {
	ID          string                 `json:"id" db:"id"`
	SourceID    *string                `json:"source_id,omitempty" db:"source_id"`
	ObjectID    *string                `json:"object_id,omitempty" db:"object_id"`
	ColumnID    *string                `json:"column_id,omitempty" db:"column_id"`
	Name        string                 `json:"name" db:"name"`
	Description *string                `json:"description,omitempty" db:"description"`
	RuleType    DQRuleType             `json:"rule_type" db:"rule_type"`
	RuleConfig  map[string]interface{} `json:"rule_config" db:"rule_config"`
	Severity    DQSeverity             `json:"severity" db:"severity"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// DQRuleFilter 规则过滤器
type DQRuleFilter struct {
	SourceID *string    `json:"source_id,omitempty" form:"source_id"`
	ObjectID *string    `json:"object_id,omitempty" form:"object_id"`
	ColumnID *string    `json:"column_id,omitempty" form:"column_id"`
	RuleType *DQRuleType `json:"rule_type,omitempty" form:"rule_type"`
	IsActive *bool      `json:"is_active,omitempty" form:"is_active"`
}

// DQRuleRequest 创建/更新规则请求
type DQRuleRequest struct {
	SourceID    *string                `json:"source_id,omitempty"`
	ObjectID    *string                `json:"object_id,omitempty"`
	ColumnID    *string                `json:"column_id,omitempty"`
	Name        string                 `json:"name" binding:"required,max=255"`
	Description *string                `json:"description,omitempty"`
	RuleType    DQRuleType             `json:"rule_type" binding:"required,oneof=not_null unique regex range enum custom_sql referential"`
	RuleConfig  map[string]interface{} `json:"rule_config"`
	Severity    DQSeverity             `json:"severity" binding:"omitempty,oneof=error warning info"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

// DQResult 数据质量检测结果
type DQResult struct {
	ID            string                 `json:"id" db:"id"`
	RuleID        string                 `json:"rule_id" db:"rule_id"`
	CheckBatchID  string                 `json:"check_batch_id" db:"check_batch_id"`
	ColumnID      *string                `json:"column_id,omitempty" db:"column_id"`
	Status        DQResultStatus         `json:"status" db:"status"`
	TotalRows     int64                  `json:"total_rows" db:"total_rows"`
	FailedRows    int64                  `json:"failed_rows" db:"failed_rows"`
	PassRate      float64                `json:"pass_rate" db:"pass_rate"`
	SampleErrors  []map[string]interface{} `json:"sample_errors,omitempty" db:"sample_errors"`
	ErrorMessage  *string                `json:"error_message,omitempty" db:"error_message"`
	CheckedAt     time.Time              `json:"checked_at" db:"checked_at"`
}

// DQCheckRequest 执行数据质量检查请求
type DQCheckRequest struct {
	RuleIDs      []string `json:"rule_ids,omitempty"`
	SourceID     *string  `json:"source_id,omitempty"`
	ObjectID     *string  `json:"object_id,omitempty"`
	ColumnID     *string  `json:"column_id,omitempty"`
	CheckAll     bool     `json:"check_all"`
	SampleLimit  int      `json:"sample_limit"` // 错误样本数量限制
}

// DQCheckResponse 数据质量检查响应
type DQCheckResponse struct {
	BatchID     string       `json:"batch_id"`
	TotalRules  int          `json:"total_rules"`
	PassedRules int          `json:"passed_rules"`
	FailedRules int          `json:"failed_rules"`
	Results     []*DQResult  `json:"results"`
	CheckedAt   time.Time    `json:"checked_at"`
}

// DQStats 数据质量统计
type DQStats struct {
	TotalRules    int     `json:"total_rules"`
	ActiveRules   int     `json:"active_rules"`
	TotalChecks   int64   `json:"total_checks"`
	PassedChecks  int64   `json:"passed_checks"`
	FailedChecks  int64   `json:"failed_checks"`
	OverallPassRate float64 `json:"overall_pass_rate"`
}

// DQRuleWithResult 规则及其最新结果
type DQRuleWithResult struct {
	DQRule
	LatestResult *DQResult `json:"latest_result,omitempty"`
}

// DQHistogramItem 数据质量历史直方图项
type DQHistogramItem struct {
	Date       string  `json:"date"`
	Total      int64   `json:"total"`
	Passed     int64   `json:"passed"`
	Failed     int64   `json:"failed"`
	PassRate   float64 `json:"pass_rate"`
}

// RuleConfig 各类规则的配置结构

// NotNullRuleConfig 非空检查配置
type NotNullRuleConfig struct{}

// UniqueRuleConfig 唯一性检查配置
type UniqueRuleConfig struct{}

// RegexRuleConfig 正则表达式检查配置
type RegexRuleConfig struct {
	Pattern string `json:"pattern"`
	Flags   string `json:"flags,omitempty"`
}

// RangeRuleConfig 范围检查配置
type RangeRuleConfig struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// EnumRuleConfig 枚举值检查配置
type EnumRuleConfig struct {
	Values []string `json:"values"`
}

// CustomSQLRuleConfig 自定义SQL检查配置
type CustomSQLRuleConfig struct {
	SQL string `json:"sql"`
}

// ReferentialRuleConfig 引用完整性检查配置
type ReferentialRuleConfig struct {
	RefSourceID *string `json:"ref_source_id,omitempty"`
	RefObjectID string  `json:"ref_object_id"`
	RefColumnID string  `json:"ref_column_id"`
}
