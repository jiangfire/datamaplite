# DataMap-Lite 项目总结

## 项目概述

DataMap-Lite 是一个轻量级数据目录系统，面向半导体显示研发部门的元数据治理解决方案。核心定位是解决"同义不同名"（如 PanelID/plt_no/玻璃编号）的数据一致性问题。

**开发周期**: Phase 1-5 (5个阶段)
**技术栈**: Go 1.25 + Gin + PostgreSQL/SQLite + React 19 + TypeScript

---

## 已完成功能清单

### Phase 1: 元数据基座 ✅
- [x] 数据源管理 (CRUD)
- [x] MySQL 采集器
- [x] MongoDB 采集器
- [x] Schema 浏览器
- [x] AES-256-GCM 加密连接串

### Phase 2: 搜索与映射 ✅
- [x] 全局字段搜索 (`GET /api/v1/columns/search`)
- [x] 字段映射管理
- [x] 血缘查询 (递归CTE)
- [x] 影响分析
- [x] Schema 变更审计

### Phase 3: 术语与DDL ✅
- [x] 业务术语管理 (CRUD)
- [x] 字段术语分配
- [x] DDL生成 (MySQL/PostgreSQL)
- [x] 数据类型映射

### Phase 4: React前端 ✅
- [x] 数据源管理页面
- [x] Schema浏览器 (树形展示)
- [x] 字段搜索页面
- [x] 字段详情页面
- [x] 血缘可视化
- [x] 业务术语管理
- [x] 响应式设计

### Phase 5: CI/CD ✅
- [x] GitHub Actions CI工作流
- [x] GitHub Actions CD工作流
- [x] Dockerfile (前后端)
- [x] docker-compose.yml
- [x] Makefile
- [x] 部署文档

---

## 项目结构

```
datamap-lite/
├── cmd/datamap/              # 应用入口
│   └── main.go
├── internal/
│   ├── api/                  # HTTP handlers
│   │   ├── handler.go
│   │   ├── router.go         # 路由定义
│   │   ├── source_handler.go
│   │   ├── schema_handler.go
│   │   └── term_handler.go
│   ├── service/              # 业务逻辑层
│   │   ├── source.go
│   │   ├── metadata.go
│   │   ├── term.go
│   │   └── ddl.go
│   ├── store/                # 数据访问层
│   │   ├── store.go          # Store接口
│   │   ├── source.go
│   │   ├── schema.go
│   │   ├── column.go
│   │   ├── mapping.go        # 字段映射
│   │   ├── lineage.go        # 血缘查询
│   │   ├── term.go           # 业务术语
│   │   ├── sqlite_*.go       # SQLite实现
│   │   └── postgres_*.go     # PostgreSQL实现
│   ├── scanner/              # 元数据采集器
│   │   ├── scanner.go        # Scanner接口
│   │   ├── mysql.go          # MySQL采集器
│   │   └── mongodb.go        # MongoDB采集器
│   ├── model/                # 数据模型
│   │   ├── models.go
│   │   └── dto.go            # API DTO
│   ├── config/               # 配置管理
│   │   └── config.go
│   └── crypto/               # 加密工具
│       └── aes.go
├── web/                      # React前端
│   ├── src/
│   │   ├── types/            # TypeScript类型
│   │   ├── services/         # API服务
│   │   ├── hooks/            # React Hooks
│   │   ├── components/       # UI组件
│   │   ├── pages/            # 页面组件
│   │   ├── App.tsx
│   │   └── index.tsx
│   ├── package.json
│   ├── Dockerfile
│   └── nginx.conf
├── migrations/               # 数据库迁移
│   └── 001_init_schema.sql
├── docs/                     # 文档
│   └── DEPLOYMENT.md
├── configs/                  # 配置文件
│   └── config.yaml
├── scripts/                  # 脚本
│   └── release.sh
├── .github/workflows/        # GitHub Actions
│   ├── ci.yml
│   └── cd.yml
├── Dockerfile                # 后端Dockerfile
├── docker-compose.yml        # Docker Compose配置
├── Makefile                  # 开发命令
├── .env.example              # 环境变量模板
├── README.md                 # 项目说明
└── PROJECT_SUMMARY.md        # 本文件
```

---

## API端点汇总

### 数据源管理
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/sources | 列出数据源 |
| POST | /api/v1/sources | 创建数据源 |
| GET | /api/v1/sources/:id | 获取数据源详情 |
| PUT | /api/v1/sources/:id | 更新数据源 |
| DELETE | /api/v1/sources/:id | 删除数据源 |
| POST | /api/v1/sources/:id/test | 测试连接 |
| POST | /api/v1/sources/:id/sync | 触发同步 |
| GET | /api/v1/sources/:id/schema | 获取Schema树 |
| GET | /api/v1/sources/:id/changes | 获取变更历史 |

### 字段管理
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/columns/search | 全局搜索字段 |
| GET | /api/v1/columns/:id | 获取字段详情 |
| GET | /api/v1/columns/:id/mappings | 获取字段映射 |
| POST | /api/v1/columns/:id/mappings | 创建字段映射 |
| DELETE | /api/v1/columns/:id/mappings/:mappingId | 删除字段映射 |
| GET | /api/v1/columns/:id/lineage | 获取血缘关系 |
| GET | /api/v1/columns/:id/impact | 获取影响分析 |
| POST | /api/v1/columns/:id/term | 分配术语到字段 |

### 业务术语
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/terms | 列出业务术语 |
| POST | /api/v1/terms | 创建业务术语 |
| GET | /api/v1/terms/:id | 获取业务术语 |
| PUT | /api/v1/terms/:id | 更新业务术语 |
| DELETE | /api/v1/terms/:id | 删除业务术语 |

### DDL生成
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/ddl/generate | 生成DDL |

---

## 数据库表结构

1. **data_sources** - 数据源注册
2. **business_terms** - 业务术语表
3. **schema_objects** - 物理对象（表/视图/集合）
4. **columns** - 字段元数据
5. **column_mappings** - 跨系统字段映射
6. **lineage_edges** - 数据血缘
7. **schema_changes** - Schema变更审计

---

## 测试覆盖率

| 包 | 覆盖率 | 状态 |
|---|---|---|
| crypto | 82.9% | ✅ |
| service | 32.0% | 🟡 |
| store | 21.0% | 🟡 |

---

## 启动命令

### 开发环境

```bash
# 后端
go run ./cmd/datamap

# 前端
cd web && pnpm dev
```

### 生产环境 (Docker)

```bash
make docker-up
```

### 测试

```bash
# 后端测试
go test ./...

# 前端测试
cd web && pnpm test
```

---

## 关键配置

### 环境变量

```bash
# 数据库 (SQLite - 开发)
DATABASE_TYPE=sqlite
DATABASE_PATH=./datamap.db

# 数据库 (PostgreSQL - 生产)
DATABASE_TYPE=postgres
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=datamap
DATABASE_USER=datamap
DATABASE_PASSWORD=yourpassword

# 安全 (必须32字节)
DATAMAP_ENCRYPTION_KEY=your-32-byte-encryption-key!!

# 服务器
SERVER_PORT=8080
LOG_LEVEL=info
```

---

## 后续扩展建议

### Phase 6+ 可选方向

1. **更多数据源**
   - Oracle 采集器
   - SQL Server 采集器
   - PostgreSQL 采集器
   - 支持更多数据库类型

2. **数据质量**
   - PII 自动识别
   - 数据分类分级
   - 数据质量规则

3. **集成能力**
   - Airflow 集成
   - BI工具集成
   - 消息通知 (钉钉/企业微信)

4. **安全增强**
   - RBAC 权限管理
   - OAuth2/LDAP 登录
   - 操作审计日志

5. **性能优化**
   - Redis 缓存
   - 全文搜索 (Elasticsearch)
   - 批量导入导出

---

## 重要文件位置

- **主入口**: `cmd/datamap/main.go`
- **路由定义**: `internal/api/router.go`
- **Store接口**: `internal/store/store.go`
- **前端入口**: `web/src/App.tsx`
- **API服务**: `web/src/services/`
- **部署配置**: `docker-compose.yml`
- **CI/CD**: `.github/workflows/`

---

## 开发团队

- **创建日期**: 2026-03-08
- **技术栈**: Go + React
- **许可证**: MIT

---

*文档版本: 1.0.0*
*最后更新: 2026-03-08*
