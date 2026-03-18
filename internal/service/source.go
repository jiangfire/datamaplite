package service

import (
	"context"
	"encoding/json"
	"fmt"

	"git.neolidy.top/neo/fuckcmdb/internal/crypto"
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
)

// SourceService 数据源服务
type SourceService struct {
	store        store.Store
	cipher       *crypto.Cipher
	registry     *scanner.Registry
	alertService *AlertService
}

// NewSourceService 创建数据源服务
func NewSourceService(store store.Store, cipher *crypto.Cipher, registry *scanner.Registry) *SourceService {
	return &SourceService{
		store:    store,
		cipher:   cipher,
		registry: registry,
	}
}

// SetAlertService 设置告警服务（用于解决循环依赖）
func (s *SourceService) SetAlertService(alertService *AlertService) {
	s.alertService = alertService
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

	if err := s.store.CreateDataSource(ctx, create); err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	// 返回创建后的数据源
	sources, err := s.store.ListDataSources(ctx)
	if err != nil {
		return nil, err
	}

	if len(sources) > 0 {
		return s.toSourceResponse(sources[0]), nil
	}

	return nil, fmt.Errorf("source created but not found")
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

// TriggerSync 触发同步
func (s *SourceService) TriggerSync(ctx context.Context, sourceID string) error {
	src, err := s.store.GetDataSource(ctx, sourceID)
	if err != nil {
		return err
	}

	// 解密连接配置
	configJSON, err := s.cipher.Decrypt(src.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to decrypt connection config: %w", err)
	}

	connConfig, err := scanner.ConnectionConfigFromJSON(configJSON)
	if err != nil {
		return err
	}

	// 获取扫描器
	sc, err := s.registry.Get(src.Type)
	if err != nil {
		return err
	}

	// 更新状态为同步中
	if err := s.store.UpdateDataSourceSyncStatus(ctx, sourceID, "syncing", nil); err != nil {
		return err
	}

	// 异步执行同步（实际应用中应该使用后台任务队列）
	go func() {
		bgCtx := context.Background()
		schemaInfo, err := sc.ScanSchema(bgCtx, *connConfig)
		if err != nil {
			errMsg := err.Error()
			s.store.UpdateDataSourceSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}

		// 保存扫描结果
		if err := s.saveSchema(bgCtx, sourceID, schemaInfo); err != nil {
			errMsg := err.Error()
			s.store.UpdateDataSourceSyncStatus(bgCtx, sourceID, "error", &errMsg)
			return
		}

		s.store.UpdateDataSourceSyncStatus(bgCtx, sourceID, "active", nil)
	}()

	return nil
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
			}

			for oldColName, oldCol := range oldColumns {
				if _, exists := newColumns[oldColName]; exists {
					continue
				}
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
			}

			// 使用对象稳定 ID，避免对象级规则和告警失效
			if err := txStore.DeleteColumnsByObject(ctx, objectID); err != nil {
				return err
			}

			for _, col := range obj.Columns {
				colCreate := &store.ColumnCreate{
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
				}

				if err := txStore.CreateColumn(ctx, colCreate); err != nil {
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
				ChangeType: "drop_object",
				ObjectType: "object",
				ObjectName: oldObj.Name,
				OldValue:   &oldValue,
			}); err != nil {
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
			go s.alertService.ProcessSchemaChange(context.Background(), change)
		}
	}

	return nil
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
	if err := txStore.CreateSchemaChange(ctx, change); err != nil {
		return err
	}

	changes, err := txStore.ListSchemaChangesBySource(ctx, change.SourceID, 1)
	if err != nil || len(changes) == 0 {
		return err
	}

	latest := changes[0]
	*detected = append(*detected, &SchemaChangeInfo{
		ID:         latest.ID,
		SourceID:   latest.SourceID,
		ObjectID:   latest.ObjectID,
		ChangeType: latest.ChangeType,
		ObjectType: latest.ObjectType,
		ObjectName: latest.ObjectName,
		OldValue:   latest.OldValue,
		NewValue:   latest.NewValue,
		DetectedAt: latest.DetectedAt,
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
		"type=%s full=%s nullable=%t pk=%t unique=%t default=%v parent=%v desc=%v",
		col.DataType, col.FullDataType, col.IsNullable, col.IsPrimaryKey, col.IsUnique, col.DefaultValue, col.ParentColumnPath, col.Description,
	)
}

func (s *SourceService) columnSignatureFromStore(col *store.ColumnRow) string {
	return fmt.Sprintf(
		"type=%s full=%s nullable=%t pk=%t unique=%t default=%v parent=%v desc=%v",
		col.DataType, col.FullDataType, col.IsNullable, col.IsPrimaryKey, col.IsUnique, col.DefaultValue, col.ParentColumnPath, col.Description,
	)
}
