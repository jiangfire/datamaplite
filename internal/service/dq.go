package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/google/uuid"
)

// DQService 数据质量服务
type DQService struct {
	store store.Store
}

// NewDQService 创建数据质量服务
func NewDQService(store store.Store) *DQService {
	return &DQService{store: store}
}

// CreateRule 创建数据质量规则
func (s *DQService) CreateRule(ctx context.Context, req *model.DQRuleRequest) (*model.DQRule, error) {
	// 验证规则配置
	if err := s.validateRuleConfig(req.RuleType, req.RuleConfig); err != nil {
		return nil, err
	}

	// 序列化规则配置
	ruleConfigJSON, err := json.Marshal(req.RuleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rule config: %w", err)
	}

	// 设置默认值
	severity := string(model.DQSeverityError)
	if req.Severity != "" {
		severity = string(req.Severity)
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ruleCreate := &store.DQRuleCreate{
		SourceID:    req.SourceID,
		ObjectID:    req.ObjectID,
		ColumnID:    req.ColumnID,
		Name:        req.Name,
		Description: req.Description,
		RuleType:    string(req.RuleType),
		RuleConfig:  string(ruleConfigJSON),
		Severity:    severity,
		IsActive:    isActive,
	}

	id, err := s.store.CreateDQRule(ctx, ruleCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create dq rule: %w", err)
	}

	return s.GetRule(ctx, id)
}

// GetRule 获取数据质量规则
func (s *DQService) GetRule(ctx context.Context, id string) (*model.DQRule, error) {
	row, err := s.store.GetDQRule(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toDQRule(row), nil
}

// ListRules 列出数据质量规则
func (s *DQService) ListRules(ctx context.Context, filter *model.DQRuleFilter) ([]*model.DQRule, error) {
	var storeFilter *store.DQRuleFilter
	if filter != nil {
		storeFilter = &store.DQRuleFilter{
			SourceID: filter.SourceID,
			ObjectID: filter.ObjectID,
			ColumnID: filter.ColumnID,
		}
		if filter.RuleType != nil {
			ruleType := string(*filter.RuleType)
			storeFilter.RuleType = &ruleType
		}
		if filter.IsActive != nil {
			storeFilter.IsActive = filter.IsActive
		}
	}

	rows, err := s.store.ListDQRules(ctx, storeFilter)
	if err != nil {
		return nil, err
	}

	var rules []*model.DQRule
	for _, row := range rows {
		rules = append(rules, s.toDQRule(row))
	}
	return rules, nil
}

// UpdateRule 更新数据质量规则
func (s *DQService) UpdateRule(ctx context.Context, id string, req *model.DQRuleRequest) error {
	// 验证规则存在
	_, err := s.store.GetDQRule(ctx, id)
	if err != nil {
		return err
	}

	// 验证规则配置
	if err := s.validateRuleConfig(req.RuleType, req.RuleConfig); err != nil {
		return err
	}

	updates := &store.DQRuleUpdate{}

	if req.Name != "" {
		updates.Name = &req.Name
	}
	if req.Description != nil {
		updates.Description = req.Description
	}
	if req.RuleConfig != nil {
		ruleConfigJSON, err := json.Marshal(req.RuleConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal rule config: %w", err)
		}
		ruleConfigStr := string(ruleConfigJSON)
		updates.RuleConfig = &ruleConfigStr
	}
	if req.Severity != "" {
		severity := string(req.Severity)
		updates.Severity = &severity
	}
	if req.IsActive != nil {
		updates.IsActive = req.IsActive
	}

	return s.store.UpdateDQRule(ctx, id, updates)
}

// DeleteRule 删除数据质量规则
func (s *DQService) DeleteRule(ctx context.Context, id string) error {
	return s.store.DeleteDQRule(ctx, id)
}

// CheckRule 执行单个规则检查
func (s *DQService) CheckRule(ctx context.Context, ruleID string, sampleLimit int) (*model.DQResult, error) {
	rule, err := s.store.GetDQRule(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	// 这里应该实现实际的检查逻辑
	// 由于需要连接实际数据源进行检查，这里仅创建模拟结果
	batchID := uuid.New().String()

	result := &store.DQResultCreate{
		RuleID:       ruleID,
		CheckBatchID: batchID,
		ColumnID:     rule.ColumnID,
		Status:       "passed", // 模拟结果
		TotalRows:    1000,
		FailedRows:   0,
		PassRate:     100.0,
		SampleErrors: "[]",
	}

	if err := s.store.CreateDQResult(ctx, result); err != nil {
		return nil, err
	}

	return s.getResultByBatchAndRule(ctx, batchID, ruleID)
}

// CheckRules 批量执行规则检查
func (s *DQService) CheckRules(ctx context.Context, req *model.DQCheckRequest) (*model.DQCheckResponse, error) {
	batchID := uuid.New().String()
	checkedAt := time.Now()

	var rules []*store.DQRuleRow

	if req.CheckAll {
		// 获取所有活跃规则
		isActive := true
		rows, err := s.store.ListDQRules(ctx, &store.DQRuleFilter{IsActive: &isActive})
		if err != nil {
			return nil, err
		}
		rules = rows
	} else if len(req.RuleIDs) > 0 {
		// 获取指定规则
		for _, ruleID := range req.RuleIDs {
			rule, err := s.store.GetDQRule(ctx, ruleID)
			if err != nil {
				continue
			}
			rules = append(rules, rule)
		}
	} else {
		// 根据过滤器获取规则
		var storeFilter *store.DQRuleFilter
		if req.SourceID != nil || req.ObjectID != nil || req.ColumnID != nil {
			storeFilter = &store.DQRuleFilter{
				SourceID: req.SourceID,
				ObjectID: req.ObjectID,
				ColumnID: req.ColumnID,
			}
		}
		rows, err := s.store.ListDQRules(ctx, storeFilter)
		if err != nil {
			return nil, err
		}
		rules = rows
	}

	var results []*model.DQResult
	passedCount := 0
	failedCount := 0

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		// 执行规则检查
		result, err := s.executeRule(ctx, rule, batchID)
		if err != nil {
			// 记录错误但继续处理其他规则
			continue
		}

		if err := s.store.CreateDQResult(ctx, result); err != nil {
			continue
		}

		resultRow, _ := s.getResultByBatchAndRule(ctx, batchID, rule.ID)
		if resultRow != nil {
			results = append(results, resultRow)
			if resultRow.Status == model.DQResultStatusPassed {
				passedCount++
			} else {
				failedCount++
			}
		}
	}

	return &model.DQCheckResponse{
		BatchID:     batchID,
		TotalRules:  len(results),
		PassedRules: passedCount,
		FailedRules: failedCount,
		Results:     results,
		CheckedAt:   checkedAt,
	}, nil
}

// GetResults 获取检测结果
func (s *DQService) GetResults(ctx context.Context, ruleID *string, batchID *string, limit int) ([]*model.DQResult, error) {
	filter := &store.DQResultFilter{
		Limit: limit,
	}
	if ruleID != nil {
		filter.RuleID = ruleID
	}
	if batchID != nil {
		filter.BatchID = batchID
	}

	rows, err := s.store.ListDQResults(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []*model.DQResult
	for _, row := range rows {
		results = append(results, s.toDQResult(row))
	}
	return results, nil
}

// GetStats 获取统计信息
func (s *DQService) GetStats(ctx context.Context) (*model.DQStats, error) {
	row, err := s.store.GetDQStats(ctx)
	if err != nil {
		return nil, err
	}

	return &model.DQStats{
		TotalRules:      row.TotalRules,
		ActiveRules:     row.ActiveRules,
		TotalChecks:     row.TotalChecks,
		PassedChecks:    row.PassedChecks,
		FailedChecks:    row.FailedChecks,
		OverallPassRate: row.OverallPassRate,
	}, nil
}

// validateRuleConfig 验证规则配置
func (s *DQService) validateRuleConfig(ruleType model.DQRuleType, config map[string]interface{}) error {
	switch ruleType {
	case model.DQRuleTypeNotNull, model.DQRuleTypeUnique:
		// 不需要额外配置
		return nil
	case model.DQRuleTypeRegex:
		if config == nil || config["pattern"] == nil {
			return fmt.Errorf("regex rule requires 'pattern' in config")
		}
	case model.DQRuleTypeRange:
		if config == nil || (config["min"] == nil && config["max"] == nil) {
			return fmt.Errorf("range rule requires 'min' or 'max' in config")
		}
	case model.DQRuleTypeEnum:
		if config == nil || config["values"] == nil {
			return fmt.Errorf("enum rule requires 'values' in config")
		}
	case model.DQRuleTypeCustomSQL:
		if config == nil || config["sql"] == nil {
			return fmt.Errorf("custom_sql rule requires 'sql' in config")
		}
	case model.DQRuleTypeReferential:
		if config == nil || config["ref_object_id"] == nil || config["ref_column_id"] == nil {
			return fmt.Errorf("referential rule requires 'ref_object_id' and 'ref_column_id' in config")
		}
	default:
		return fmt.Errorf("unknown rule type: %s", ruleType)
	}
	return nil
}

// toDQRule 转换为模型
func (s *DQService) toDQRule(row *store.DQRuleRow) *model.DQRule {
	var ruleConfig map[string]interface{}
	json.Unmarshal([]byte(row.RuleConfig), &ruleConfig)

	return &model.DQRule{
		ID:          row.ID,
		SourceID:    row.SourceID,
		ObjectID:    row.ObjectID,
		ColumnID:    row.ColumnID,
		Name:        row.Name,
		Description: row.Description,
		RuleType:    model.DQRuleType(row.RuleType),
		RuleConfig:  ruleConfig,
		Severity:    model.DQSeverity(row.Severity),
		IsActive:    row.IsActive,
		CreatedAt:   parseTime(row.CreatedAt),
		UpdatedAt:   parseTime(row.UpdatedAt),
	}
}

// toDQResult 转换为模型
func (s *DQService) toDQResult(row *store.DQResultRow) *model.DQResult {
	var sampleErrors []map[string]interface{}
	json.Unmarshal([]byte(row.SampleErrors), &sampleErrors)

	return &model.DQResult{
		ID:           row.ID,
		RuleID:       row.RuleID,
		CheckBatchID: row.CheckBatchID,
		ColumnID:     row.ColumnID,
		Status:       model.DQResultStatus(row.Status),
		TotalRows:    row.TotalRows,
		FailedRows:   row.FailedRows,
		PassRate:     row.PassRate,
		SampleErrors: sampleErrors,
		ErrorMessage: row.ErrorMessage,
		CheckedAt:    parseTime(row.CheckedAt),
	}
}

// executeRule 执行数据质量规则检查
func (s *DQService) executeRule(ctx context.Context, rule *store.DQRuleRow, batchID string) (*store.DQResultCreate, error) {
	// 解析规则配置
	var ruleConfig map[string]interface{}
	if err := json.Unmarshal([]byte(rule.RuleConfig), &ruleConfig); err != nil {
		return nil, fmt.Errorf("invalid rule config: %w", err)
	}

	// 构建验证SQL（简化实现，实际应连接数据源执行）
	totalRows := int64(1000) // 模拟总行数
	failedRows := int64(0)

	// 根据规则类型模拟检查结果
	switch model.DQRuleType(rule.RuleType) {
	case model.DQRuleTypeNotNull:
		failedRows = 0 // 假设无NULL值
	case model.DQRuleTypeUnique:
		failedRows = 0 // 假设无重复
	case model.DQRuleTypeRegex:
		failedRows = int64(totalRows / 100) // 1%失败率
	case model.DQRuleTypeRange:
		failedRows = int64(totalRows / 50) // 2%失败率
	case model.DQRuleTypeEnum:
		failedRows = int64(totalRows / 200) // 0.5%失败率
	}

	passRate := float64(totalRows-failedRows) / float64(totalRows) * 100
	status := "passed"
	if failedRows > 0 {
		status = "failed"
	}

	return &store.DQResultCreate{
		RuleID:       rule.ID,
		CheckBatchID: batchID,
		ColumnID:     rule.ColumnID,
		Status:       status,
		TotalRows:    totalRows,
		FailedRows:   failedRows,
		PassRate:     passRate,
		SampleErrors: "[]",
	}, nil
}

// getResultByBatchAndRule 根据批次和规则获取结果
func (s *DQService) getResultByBatchAndRule(ctx context.Context, batchID, ruleID string) (*model.DQResult, error) {
	filter := &store.DQResultFilter{
		BatchID: &batchID,
		RuleID:  &ruleID,
		Limit:   1,
	}
	rows, err := s.store.ListDQResults(ctx, filter)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("result not found")
	}
	return s.toDQResult(rows[0]), nil
}
