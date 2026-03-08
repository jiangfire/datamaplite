# DataMap-Lite

轻量级数据目录系统，面向半导体显示研发部门的元数据治理解决方案。解决"同义不同名"（如 PanelID/plt_no/玻璃编号）的数据一致性问题。

## 功能特性

- **多数据源支持**: MySQL, PostgreSQL, MongoDB, Oracle, SQL Server
- **Schema 浏览器**: 可视化展示表结构和字段信息
- **全局搜索**: 跨数据源字段搜索，支持模糊匹配
- **字段映射**: 建立同义不同名字段之间的关系
- **数据血缘**: 追踪数据流转路径，分析字段依赖
- **影响分析**: 评估字段变更的影响范围
- **业务术语**: 数据标准化管理
- **DDL 生成**: 支持 MySQL 和 PostgreSQL 的 DDL 导出

## 快速开始

### Docker Compose (推荐)

```bash
# 克隆仓库
git clone https://github.com/yourusername/datamap-lite.git
cd datamap-lite

# 配置环境
cp .env.example .env
# 编辑 .env 设置数据库密码和加密密钥

# 启动服务
make docker-up
```

访问:
- Web UI: http://localhost
- API: http://localhost:8080
- API 文档: http://localhost:8080/swagger/index.html

## 开发

### 后端开发

```bash
# 启动后端服务
go run ./cmd/datamap

# 运行测试
go test ./...

# 代码检查
golangci-lint run
```

### 前端开发

```bash
cd web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建
pnpm build
```

## Makefile 命令

```bash
make help          # 显示所有可用命令
make dev           # 同时启动前后端开发服务器
make docker-up     # 使用 Docker Compose 启动服务
make docker-down   # 停止 Docker 服务
make test          # 运行所有测试
make lint          # 运行代码检查
make build         # 构建所有二进制文件
make clean         # 清理构建产物
```

## 部署

详见 [部署文档](docs/DEPLOYMENT.md)

### 生产环境部署

```bash
# 使用 Docker Compose
docker-compose up -d

# 或使用预构建镜像
wget https://raw.githubusercontent.com/yourusername/datamap-lite/main/docker-compose.yml
docker-compose up -d
```

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   React Frontend │────▶│   Go Backend    │────▶│   PostgreSQL   │
│   (Rsbuild)     │     │   (Gin)         │     │   (Metadata)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │  Target DBs     │
                        │  MySQL/MongoDB  │
                        └─────────────────┘
```

## 项目结构

```
datamap-lite/
├── cmd/datamap/          # 应用入口
├── internal/
│   ├── api/              # HTTP handlers
│   ├── service/          # 业务逻辑
│   ├── store/            # 数据访问层
│   ├── scanner/          # 元数据采集器
│   ├── model/            # 数据模型
│   ├── config/           # 配置管理
│   └── crypto/           # 加密工具
├── web/                  # React 前端
├── migrations/           # 数据库迁移
├── docs/                 # 文档
└── docker-compose.yml    # Docker 部署配置
```

## CI/CD

- **CI**: GitHub Actions 自动运行测试、代码检查和构建
- **CD**: 自动构建 Docker 镜像并推送到 Docker Hub
- **发布**: 打 tag 自动创建 GitHub Release

## 技术栈

**后端:**
- Go 1.25
- Gin Web Framework
- PostgreSQL (pgx)
- golang-migrate
- Zap (日志)
- Viper (配置)

**前端:**
- React 19
- TypeScript
- Rsbuild
- Tailwind CSS 4
- React Router 7

**运维:**
- Docker & Docker Compose
- GitHub Actions
- Makefile

## 贡献

欢迎提交 Issue 和 Pull Request!

## 许可证

MIT License

## 联系方式

- GitHub: https://github.com/yourusername/datamap-lite
- Email: your.email@example.com
