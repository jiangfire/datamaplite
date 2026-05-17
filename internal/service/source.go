package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jiangfire/datamaplite/internal/crypto"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/store"
)

// SourceService 数据源服务
type SourceService struct {
	store                  store.Store
	cipher                 *crypto.Cipher
	registry               *scanner.Registry
	alertService           *AlertService
	governanceEventService *GovernanceEventService
	ownerID                string
	syncLeaseTTL           time.Duration
	syncLeaseStaleAfter    time.Duration
	syncMu                 sync.Mutex
	syncInFlight           map[string]struct{}
}

// NewSourceService 创建数据源服务
func NewSourceService(store store.Store, cipher *crypto.Cipher, registry *scanner.Registry) *SourceService {
	return &SourceService{
		store:               store,
		cipher:              cipher,
		registry:            registry,
		ownerID:             "sync_owner_" + uuid.NewString(),
		syncLeaseTTL:        10 * time.Second,
		syncLeaseStaleAfter: 20 * time.Second,
		syncInFlight:        make(map[string]struct{}),
	}
}

// SetAlertService 设置告警服务（用于解决循环依赖）
func (s *SourceService) SetAlertService(alertService *AlertService) {
	s.alertService = alertService
}

// SetGovernanceEventService 设置治理事件发送服务。
func (s *SourceService) SetGovernanceEventService(governanceEventService *GovernanceEventService) {
	s.governanceEventService = governanceEventService
}

// CreateSource 创建数据源
func (s *SourceService) CreateSource(ctx context.Context, req *model.CreateSourceRequest) (*model.SourceResponse, error) {
	// 构建连接配置
	connConfig := scanner.ConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		Database: req.Database,
		Username: req.Username,
		Password: req.Password,
		SSLMode:  req.SSLMode,
	}

	// 加密连接配置
	configJSON, err := connConfig.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection config: %w", err)
	}

	encryptedConfig, err := s.cipher.Encrypt(configJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt connection config: %w", err)
	}

	// 创建数据源记录
	var descPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}

	create := &store.DataSourceCreate{
		Name:             req.Name,
		Description:      descPtr,
		Type:             string(req.Type),
		Host:             req.Host,
		Port:             req.Port,
		Database:         req.Database,
		ConnectionConfig: encryptedConfig,
	}

	sourceID, err := s.store.CreateDataSource(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	src, err := s.store.GetDataSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source created but not found: %w", err)
	}

	return s.toSourceResponse(src), nil
}

// ListSources 列出所有数据源
func (s *SourceService) ListSources(ctx context.Context) ([]*model.SourceListItem, error) {
	sources, err := s.store.ListDataSources(ctx)
	if err != nil {
		return nil, err
	}

	var items []*model.SourceListItem
	for _, src := range sources {
		items = append(items, s.toSourceListItem(src))
	}
	return items, nil
}

// GetSource 获取数据源详情
func (s *SourceService) GetSource(ctx context.Context, id string) (*model.SourceResponse, error) {
	src, err := s.store.GetDataSource(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toSourceResponse(src), nil
}

// UpdateSource 更新数据源
func (s *SourceService) UpdateSource(ctx context.Context, id string, req *model.UpdateSourceRequest) error {
	updates := &store.DataSourceUpdate{}

	if req.Name != "" {
		updates.Name = &req.Name
	}
	if req.Description != "" {
		updates.Description = &req.Description
	}
	if req.Host != "" {
		updates.Host = &req.Host
	}
	if req.Port != 0 {
		updates.Port = &req.Port
	}
	if req.Database != "" {
		updates.Database = &req.Database
	}

	// 如果有密码相关更新，需要重新加密连接配置
	if req.Password != "" || req.Username != "" || req.Host != "" || req.Port != 0 || req.Database != "" {
		// 获取现有记录来填充未提供的字段
		existing, err := s.store.GetDataSource(ctx, id)
		if err != nil {
			return err
		}

		// 解密现有配置
		configJSON, err := s.cipher.Decrypt(existing.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to decrypt connection config: %w", err)
		}

		var connConfig scanner.ConnectionConfig
		if err := json.Unmarshal([]byte(configJSON), &connConfig); err != nil {
			return fmt.Errorf("failed to unmarshal connection config: %w", err)
		}

		// 更新提供的字段
		if req.Username != "" {
			connConfig.Username = req.Username
		}
		if req.Password != "" {
			connConfig.Password = req.Password
		}
		if req.Host != "" {
			connConfig.Host = req.Host
		}
		if req.Port != 0 {
			connConfig.Port = req.Port
		}
		if req.Database != "" {
			connConfig.Database = req.Database
		}

		// 重新加密
		newConfigJSON, _ := connConfig.ToJSON()
		encryptedConfig, err := s.cipher.Encrypt(newConfigJSON)
		if err != nil {
			return fmt.Errorf("failed to encrypt connection config: %w", err)
		}
		updates.ConnectionConfig = &encryptedConfig
	}

	return s.store.UpdateDataSource(ctx, id, updates)
}

// DeleteSource 删除数据源
func (s *SourceService) DeleteSource(ctx context.Context, id string) error {
	return s.store.DeleteDataSource(ctx, id)
}

// TestConnection 测试连接
func (s *SourceService) TestConnection(ctx context.Context, dbType string, config scanner.ConnectionConfig) error {
	sc, err := s.registry.Get(dbType)
	if err != nil {
		return err
	}
	return sc.TestConnection(ctx, config)
}

// GetSyncLease 获取数据源当前同步租约。
func (s *SourceService) GetSyncLease(ctx context.Context, sourceID string) (*store.SyncLeaseRow, error) {
	return s.store.GetSyncLease(ctx, sourceID)
}

// ForceReleaseStaleSyncLease 释放陈旧同步租约，避免实例异常退出后只能等待 TTL。
func (s *SourceService) ForceReleaseStaleSyncLease(ctx context.Context, sourceID string, staleAfter time.Duration) error {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if staleAfter <= 0 {
		staleAfter = s.syncLeaseStaleAfter
	}
	if staleAfter <= 0 {
		staleAfter = 20 * time.Second
	}

	lease, err := s.store.GetSyncLease(ctx, sourceID)
	if err != nil {
		return err
	}
	if lease == nil {
		return nil
	}

	updatedAt := parseTime(lease.UpdatedAt)
	if updatedAt.IsZero() {
		return fmt.Errorf("invalid sync lease updated_at for source %s: %s", sourceID, lease.UpdatedAt)
	}
	if time.Since(updatedAt.UTC()) < staleAfter {
		return fmt.Errorf("sync lease still fresh for source %s", sourceID)
	}

	return s.store.ForceReleaseSyncLease(ctx, sourceID)
}

// TriggerSync 触发同步
func (s *SourceService) TriggerSync(ctx context.Context, sourceID string) error {
	src, err := s.store.GetDataSource(ctx, sourceID)
	if err != nil {
		return err
	}

	acquired, err := s.beginSync(ctx, sourceID)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("source sync already in progress: %s", sourceID)
	}

	// 解密连接配置
	configJSON, err := s.cipher.Decrypt(src.ConnectionConfig)
	if err != nil {
		s.endSync(sourceID)
		return fmt.Errorf("failed to decrypt connection config: %w", err)
	}

	connConfig, err := scanner.ConnectionConfigFromJSON(configJSON)
	if err != nil {
		s.endSync(sourceID)
		return err
	}

	// 获取扫描器
	sc, err := s.registry.Get(src.Type)
	if err != nil {
		s.endSync(sourceID)
		return err
	}

	// 更新状态为同步中
	if err := s.store.UpdateDataSourceSyncStatus(ctx, sourceID, "syncing", nil); err != nil {
		s.endSync(sourceID)
		return err
	}

	auditMeta := GovernanceAuditMetaFromContext(ctx)

	// 异步执行同步（实际应用中应该使用后台任务队列）
	go func() {
		defer s.endSync(sourceID)

		stopLeaseRenewal := s.startSyncLeaseRenewal(sourceID)
		defer stopLeaseRenewal()

		bgCtx := WithGovernanceAuditMeta(context.Background(), auditMeta)
		schemaInfo, err := sc.ScanSchema(bgCtx, *connConfig)
		if err != nil {
			errMsg := err.Error()
			s.updateSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}
		leaseOwned, err := s.syncLeaseOwned(bgCtx, sourceID)
		if err != nil {
			errMsg := err.Error()
			s.updateSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}
		if !leaseOwned {
			s.handleLostSyncLease(bgCtx, sourceID)
			return
		}

		// 保存扫描结果
		if err := s.saveSchema(bgCtx, sourceID, schemaInfo); err != nil {
			errMsg := err.Error()
			s.updateSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}
		leaseOwned, err = s.syncLeaseOwned(bgCtx, sourceID)
		if err != nil {
			errMsg := err.Error()
			s.updateSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}
		if !leaseOwned {
			s.handleLostSyncLease(bgCtx, sourceID)
			return
		}

		s.updateSyncStatus(bgCtx, sourceID, "active", nil)
	}()

	return nil
}

func (s *SourceService) beginSync(ctx context.Context, sourceID string) (bool, error) {
	s.syncMu.Lock()
	if s.syncInFlight == nil {
		s.syncInFlight = make(map[string]struct{})
	}
	if _, exists := s.syncInFlight[sourceID]; exists {
		s.syncMu.Unlock()
		return false, nil
	}
	s.syncInFlight[sourceID] = struct{}{}
	s.syncMu.Unlock()

	now := time.Now().UTC()
	acquired, err := s.store.TryAcquireSyncLease(
		ctx,
		sourceID,
		s.ownerID,
		now.Format(time.RFC3339Nano),
		now.Add(s.syncLeaseTTL).Format(time.RFC3339Nano),
	)
	if err != nil {
		s.releaseLocalSync(sourceID)
		return false, err
	}
	if !acquired {
		s.releaseLocalSync(sourceID)
		return false, nil
	}

	return true, nil
}

func (s *SourceService) endSync(sourceID string) {
	s.releaseLocalSync(sourceID)
	_ = s.store.ReleaseSyncLease(context.Background(), sourceID, s.ownerID)
}

func (s *SourceService) releaseLocalSync(sourceID string) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	delete(s.syncInFlight, sourceID)
}

func (s *SourceService) startSyncLeaseRenewal(sourceID string) func() {
	if s.syncLeaseTTL <= 0 {
		return func() {}
	}

	stopCh := make(chan struct{})
	interval := s.syncLeaseTTL / 2
	if interval <= 0 {
		interval = time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				leaseUntil := time.Now().UTC().Add(s.syncLeaseTTL).Format(time.RFC3339Nano)
				_ = s.store.RenewSyncLease(context.Background(), sourceID, s.ownerID, leaseUntil)
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
	}
}

func (s *SourceService) syncLeaseOwned(ctx context.Context, sourceID string) (bool, error) {
	lease, err := s.store.GetSyncLease(ctx, sourceID)
	if err != nil {
		return false, err
	}
	if lease == nil {
		return false, nil
	}
	if lease.OwnerID != s.ownerID {
		return false, nil
	}
	leaseUntil := parseTime(lease.LeaseUntil)
	if leaseUntil.IsZero() {
		return false, fmt.Errorf("invalid sync lease lease_until for source %s: %s", sourceID, lease.LeaseUntil)
	}
	if !leaseUntil.After(time.Now().UTC()) {
		return false, nil
	}
	return true, nil
}

func (s *SourceService) handleLostSyncLease(ctx context.Context, sourceID string) {
	lease, err := s.store.GetSyncLease(ctx, sourceID)
	if err != nil {
		return
	}
	if lease != nil && lease.OwnerID != s.ownerID {
		leaseUntil := parseTime(lease.LeaseUntil)
		if !leaseUntil.IsZero() && leaseUntil.After(time.Now().UTC()) {
			return
		}
	}
	source, err := s.store.GetDataSource(ctx, sourceID)
	if err != nil {
		return
	}
	if source == nil || source.Status != "syncing" {
		return
	}

	errMsg := fmt.Sprintf("sync lease lost for source %s", sourceID)
	s.updateSyncStatus(context.WithoutCancel(ctx), sourceID, "error", &errMsg)
}

// saveSchema 保存Schema信息
func (s *SourceService) saveSchema(ctx context.Context, sourceID string, schemaInfo *scanner.SchemaInfo) error {
	detectedChanges := make([]*SchemaChangeInfo, 0)

	err := s.store.WithTx(ctx, func(txStore store.Store) error {
		// 获取现有对象用于变更检测
		existingObjs, err := txStore.ListSchemaObjectsBySource(ctx, sourceID)
		if err != nil {
			return err
		}

		existingMap := make(map[string]*store.SchemaObjectRow)
		existingColumns := make(map[string]map[string]*store.ColumnRow)
		for _, obj := range existingObjs {
			existingMap[s.schemaObjectKey(obj.Schema, obj.Name)] = obj

			cols, err := txStore.ListColumnsByObject(ctx, obj.ID)
			if err != nil {
				return err
			}

			columnMap := make(map[string]*store.ColumnRow, len(cols))
			for _, col := range cols {
				columnMap[col.Name] = col
			}
			existingColumns[obj.ID] = columnMap
		}

		currentKeys := make(map[string]struct{}, len(schemaInfo.Objects))

		// 创建或更新对象
		for _, obj := range schemaInfo.Objects {
			objectKey := s.schemaObjectKey(obj.Schema, obj.Name)
			currentKeys[objectKey] = struct{}{}

			existingObj, exists := existingMap[objectKey]
			objectID := ""
			if exists {
				objectID = existingObj.ID
				if s.objectNeedsUpdate(existingObj, obj) {
					if err := txStore.UpdateSchemaObject(ctx, objectID, &store.SchemaObjectUpdate{
						Type:        obj.Type,
						Schema:      obj.Schema,
						Description: obj.Description,
						RowCount:    obj.RowCount,
						SizeBytes:   obj.SizeBytes,
						ColumnCount: len(obj.Columns),
					}); err != nil {
						return err
					}
				}
			} else {
				objCreate := &store.SchemaObjectCreate{
					SourceID:    sourceID,
					Name:        obj.Name,
					Type:        obj.Type,
					Schema:      obj.Schema,
					Description: obj.Description,
					RowCount:    obj.RowCount,
					SizeBytes:   obj.SizeBytes,
					ColumnCount: len(obj.Columns),
				}

				objectID, err = txStore.CreateSchemaObject(ctx, objCreate)
				if err != nil {
					return err
				}

				if err := s.recordSchemaChange(ctx, txStore, &detectedChanges, &store.SchemaChangeCreate{
					SourceID:   sourceID,
					ObjectID:   &objectID,
					ChangeType: "add_object",
					ObjectType: "object",
					ObjectName: obj.Name,
					NewValue:   &obj.Type,
				}); err != nil {
					return err
				}
			}

			oldColumns := existingColumns[objectID]
			if oldColumns == nil {
				oldColumns = map[string]*store.ColumnRow{}
			}
			newColumns := make(map[string]scanner.ColumnInfo, len(obj.Columns))
			for _, col := range obj.Columns {
				newColumns[col.Name] = col
			}

			for _, col := range obj.Columns {
				oldCol, exists := oldColumns[col.Name]
				if !exists {
					newValue := s.columnSignatureFromScanner(col)
					if err := s.recordSchemaChange(ctx, txStore, &detectedChanges, &store.SchemaChangeCreate{
						SourceID:   sourceID,
						ObjectID:   &objectID,
						ChangeType: "add_column",
						ObjectType: "column",
						ObjectName: fmt.Sprintf("%s.%s", obj.Name, col.Name),
						NewValue:   &newValue,
					}); err != nil {
						return err
					}

					if err := txStore.CreateColumn(ctx, s.toColumnCreate(objectID, col)); err != nil {
						return err
					}
					continue
				}

				oldValue := s.columnSignatureFromStore(oldCol)
				newValue := s.columnSignatureFromScanner(col)
				if oldValue != newValue {
					if err := s.recordSchemaChange(ctx, txStore, &detectedChanges, &store.SchemaChangeCreate{
						SourceID:   sourceID,
						ObjectID:   &objectID,
						ChangeType: "alter_column",
						ObjectType: "column",
						ObjectName: fmt.Sprintf("%s.%s", obj.Name, col.Name),
						OldValue:   &oldValue,
						NewValue:   &newValue,
					}); err != nil {
						return err
					}
				}

				if oldValue != newValue {
					if err := txStore.UpdateColumn(ctx, oldCol.ID, s.toColumnUpdate(col)); err != nil {
						return err
					}
				}

				delete(oldColumns, col.Name)
			}

			for oldColName, oldCol := range oldColumns {
				oldValue := s.columnSignatureFromStore(oldCol)
				if err := s.recordSchemaChange(ctx, txStore, &detectedChanges, &store.SchemaChangeCreate{
					SourceID:   sourceID,
					ObjectID:   &objectID,
					ChangeType: "drop_column",
					ObjectType: "column",
					ObjectName: fmt.Sprintf("%s.%s", obj.Name, oldColName),
					OldValue:   &oldValue,
				}); err != nil {
					return err
				}
				if err := txStore.DeleteLineageEdgesByNode(ctx, oldCol.ID); err != nil {
					return err
				}
				if err := txStore.DeleteColumn(ctx, oldCol.ID); err != nil {
					return err
				}
			}
		}

		// 删除已不存在的对象
		for objectKey, oldObj := range existingMap {
			if _, exists := currentKeys[objectKey]; exists {
				continue
			}

			oldValue := oldObj.Type
			if err := s.recordSchemaChange(ctx, txStore, &detectedChanges, &store.SchemaChangeCreate{
				SourceID:   sourceID,
				ObjectID:   &oldObj.ID,
				ChangeType: "drop_object",
				ObjectType: "object",
				ObjectName: oldObj.Name,
				OldValue:   &oldValue,
			}); err != nil {
				return err
			}

			for _, oldCol := range existingColumns[oldObj.ID] {
				if err := txStore.DeleteLineageEdgesByNode(ctx, oldCol.ID); err != nil {
					return err
				}
			}
			if err := txStore.DeleteLineageEdgesByNode(ctx, oldObj.ID); err != nil {
				return err
			}
			if err := txStore.DeleteSchemaObject(ctx, oldObj.ID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 触发告警
	if s.alertService != nil {
		for _, change := range detectedChanges {
			changeCopy := change
			go func() {
				ignoreError(s.alertService.ProcessSchemaChange(context.Background(), changeCopy))
			}()
		}
	}

	if s.governanceEventService != nil && s.governanceEventService.Enabled() {
		sourceName := sourceID
		if source, err := s.store.GetDataSource(ctx, sourceID); err == nil && source != nil {
			sourceName = source.Name
		}
		auditMeta := GovernanceAuditMetaFromContext(ctx)

		for _, change := range detectedChanges {
			changeCopy := *change
			go func() {
				eventCtx := WithGovernanceAuditMeta(context.Background(), auditMeta)
				if err := s.publishSchemaChangeEvent(eventCtx, sourceName, &changeCopy); err != nil {
					return
				}
			}()
		}
	}

	return nil
}

func (s *SourceService) publishSchemaChangeEvent(ctx context.Context, sourceName string, change *SchemaChangeInfo) error {
	if change == nil || s.governanceEventService == nil || !s.governanceEventService.Enabled() {
		return nil
	}

	auditMeta := GovernanceAuditMetaFromContext(ctx)
	priority := "medium"
	switch change.ChangeType {
	case "drop_object", "drop_column":
		priority = "high"
	}

	payload := map[string]interface{}{
		"title":         fmt.Sprintf("处理结构变更：%s", change.ObjectName),
		"summary":       fmt.Sprintf("数据源 [%s] 检测到 %s 变更：%s", sourceName, change.ChangeType, change.ObjectName),
		"display_name":  change.ObjectName,
		"priority":      priority,
		"change_type":   change.ChangeType,
		"source_id":     change.SourceID,
		"object_type":   change.ObjectType,
		"object_name":   change.ObjectName,
		"schema_change": change.ID,
	}
	if change.ObjectID != nil && *change.ObjectID != "" {
		payload["object_id"] = *change.ObjectID
	}
	if change.OldValue != nil {
		payload["old_value"] = *change.OldValue
	}
	if change.NewValue != nil {
		payload["new_value"] = *change.NewValue
	}

	actorID := "system"
	if auditMeta.ActorID != "" {
		actorID = auditMeta.ActorID
	}
	traceID := "schema_change_" + change.ID
	if auditMeta.TraceID != "" {
		traceID = auditMeta.TraceID
	}

	return s.governanceEventService.Publish(ctx, GovernanceEvent{
		EventID:      "schema_change_" + change.ID,
		EventType:    "metadata.schema.changed",
		OccurredAt:   change.DetectedAt,
		ResourceType: "schema_change",
		ResourceID:   change.ID,
		ActorID:      actorID,
		TraceID:      traceID,
		Payload:      payload,
	})
}

func (s *SourceService) updateSyncStatus(ctx context.Context, sourceID string, status string, errMsg *string) {
	ignoreError(s.store.UpdateDataSourceSyncStatus(ctx, sourceID, status, errMsg))
}

// toSourceResponse 转换为响应格式
func (s *SourceService) toSourceResponse(src *store.DataSourceRow) *model.SourceResponse {
	resp := &model.SourceResponse{
		ID:            src.ID,
		Name:          src.Name,
		Type:          model.DataSourceType(src.Type),
		Host:          src.Host,
		Port:          src.Port,
		Database:      src.Database,
		Status:        model.DataSourceStatus(src.Status),
		Description:   src.Description,
		LastSyncAt:    nil,
		LastSyncError: src.LastSyncError,
		CreatedAt:     src.CreatedAt,
		UpdatedAt:     src.UpdatedAt,
	}

	resp.LastSyncAt = src.LastSyncAt

	return resp
}

// toSourceListItem 转换为列表项
func (s *SourceService) toSourceListItem(src *store.DataSourceRow) *model.SourceListItem {
	return &model.SourceListItem{
		ID:            src.ID,
		Name:          src.Name,
		Description:   src.Description,
		Type:          model.DataSourceType(src.Type),
		Host:          src.Host,
		Port:          src.Port,
		Database:      src.Database,
		Status:        model.DataSourceStatus(src.Status),
		LastSyncAt:    src.LastSyncAt,
		LastSyncError: src.LastSyncError,
		CreatedAt:     src.CreatedAt,
		UpdatedAt:     src.UpdatedAt,
	}
}

func (s *SourceService) recordSchemaChange(ctx context.Context, txStore store.Store, detected *[]*SchemaChangeInfo, change *store.SchemaChangeCreate) error {
	if change.ID == "" {
		change.ID = uuid.NewString()
	}
	if change.DetectedAt == "" {
		change.DetectedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if err := txStore.CreateSchemaChange(ctx, change); err != nil {
		return err
	}

	*detected = append(*detected, &SchemaChangeInfo{
		ID:         change.ID,
		SourceID:   change.SourceID,
		ObjectID:   change.ObjectID,
		ChangeType: change.ChangeType,
		ObjectType: change.ObjectType,
		ObjectName: change.ObjectName,
		OldValue:   change.OldValue,
		NewValue:   change.NewValue,
		DetectedAt: change.DetectedAt,
	})
	return nil
}

func (s *SourceService) schemaObjectKey(schema *string, name string) string {
	if schema == nil || *schema == "" {
		return name
	}
	return *schema + "." + name
}

func (s *SourceService) columnSignatureFromScanner(col scanner.ColumnInfo) string {
	return fmt.Sprintf(
		"type=%s full=%s nullable=%t pk=%t unique=%t default=%v parent=%v desc=%v confidence=%f",
		col.DataType,
		col.FullDataType,
		col.IsNullable,
		col.IsPrimaryKey,
		col.IsUnique,
		stringValue(col.DefaultValue),
		stringValue(col.ParentColumnPath),
		stringValue(col.Description),
		s.normalizeConfidence(col.Confidence),
	)
}

func (s *SourceService) columnSignatureFromStore(col *store.ColumnRow) string {
	return fmt.Sprintf(
		"type=%s full=%s nullable=%t pk=%t unique=%t default=%v parent=%v desc=%v confidence=%f",
		col.DataType,
		col.FullDataType,
		col.IsNullable,
		col.IsPrimaryKey,
		col.IsUnique,
		stringValue(col.DefaultValue),
		stringValue(col.ParentColumnPath),
		stringValue(col.Description),
		col.Confidence,
	)
}

func (s *SourceService) objectNeedsUpdate(existing *store.SchemaObjectRow, incoming scanner.ObjectInfo) bool {
	return existing.Type != incoming.Type ||
		!stringPtrEqual(existing.Schema, incoming.Schema) ||
		!stringPtrEqual(existing.Description, incoming.Description) ||
		!int64PtrEqual(existing.RowCount, incoming.RowCount) ||
		!int64PtrEqual(existing.SizeBytes, incoming.SizeBytes) ||
		existing.ColumnCount != len(incoming.Columns)
}

func (s *SourceService) toColumnCreate(objectID string, col scanner.ColumnInfo) *store.ColumnCreate {
	return &store.ColumnCreate{
		ObjectID:         objectID,
		Name:             col.Name,
		DataType:         col.DataType,
		FullDataType:     col.FullDataType,
		IsNullable:       col.IsNullable,
		DefaultValue:     col.DefaultValue,
		IsPrimaryKey:     col.IsPrimaryKey,
		IsUnique:         col.IsUnique,
		OrdinalPosition:  col.OrdinalPosition,
		Description:      col.Description,
		ParentColumnPath: col.ParentColumnPath,
		Confidence:       s.normalizeConfidence(col.Confidence),
	}
}

func (s *SourceService) toColumnUpdate(col scanner.ColumnInfo) *store.ColumnUpdate {
	return &store.ColumnUpdate{
		DataType:         col.DataType,
		FullDataType:     col.FullDataType,
		IsNullable:       col.IsNullable,
		DefaultValue:     col.DefaultValue,
		IsPrimaryKey:     col.IsPrimaryKey,
		IsUnique:         col.IsUnique,
		OrdinalPosition:  col.OrdinalPosition,
		Description:      col.Description,
		ParentColumnPath: col.ParentColumnPath,
		Confidence:       s.normalizeConfidence(col.Confidence),
	}
}

func (s *SourceService) normalizeConfidence(confidence float64) float64 {
	if confidence <= 0 {
		return 1.0
	}
	return confidence
}

func stringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
