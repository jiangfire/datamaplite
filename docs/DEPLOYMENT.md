# DataMap-Lite 部署指南

## 快速开始

### 方式一：Docker Compose (推荐)

1. **克隆仓库**
```bash
git clone https://github.com/yourusername/datamap-lite.git
cd datamap-lite
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，设置数据库密码和加密密钥
```

3. **启动服务**
```bash
make docker-up
# 或
docker-compose up -d
```

4. **访问服务**
- Web UI: http://localhost
- API: http://localhost:8080
- pgAdmin: http://localhost:5050 (可选)

### 方式二：使用预构建镜像

```bash
# 创建 docker-compose.yml
wget https://raw.githubusercontent.com/yourusername/datamap-lite/main/docker-compose.yml
wget https://raw.githubusercontent.com/yourusername/datamap-lite/main/.env.example -O .env

# 编辑 .env 配置
echo "ENCRYPTION_KEY=your-32-byte-encryption-key-here!!" >> .env
echo "JWT_SECRET=replace-this-jwt-secret" >> .env

# 使用官方镜像
docker-compose up -d
```

### 方式三：二进制部署

1. **下载二进制文件**
从 [Releases](https://github.com/yourusername/datamap-lite/releases) 页面下载。

2. **运行**
```bash
export DATAMAP_DATABASE_TYPE=postgres
export DATAMAP_DATABASE_HOST=localhost
export DATAMAP_DATABASE_PORT=5432
export DATAMAP_DATABASE_DATABASE=datamap
export DATAMAP_DATABASE_USERNAME=datamap
export DATAMAP_DATABASE_PASSWORD=yourpassword
export DATAMAP_AUTH_JWT_SECRET=replace-this-jwt-secret
export DATAMAP_ENCRYPTION_KEY=your-32-byte-key-here!!

./datamap
```

## 生产环境部署

### 系统要求

| 组件 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | 2核 | 4核 |
| 内存 | 4GB | 8GB |
| 磁盘 | 20GB | 50GB+ SSD |
| 网络 | 10Mbps | 100Mbps |

### 安全配置

1. **修改默认密码**
```bash
# 在 .env 文件中设置强密码
DB_PASSWORD=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 24 | head -c 32)
JWT_SECRET=$(openssl rand -base64 32)
```

2. **使用 HTTPS**
```yaml
# docker-compose.yml 中添加 nginx-proxy
services:
  nginx-proxy:
    image: nginxproxy/nginx-proxy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - ./certs:/etc/nginx/certs
```

3. **配置防火墙**
```bash
# 仅开放必要端口
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

### 数据库备份

```bash
# 创建备份脚本
cat > backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker-compose exec -T postgres pg_dump -U datamap datamap > backup_$DATE.sql
gzip backup_$DATE.sql
# 上传到 S3 或其他存储
aws s3 cp backup_$DATE.sql.gz s3://your-bucket/datamap-backups/
EOF
chmod +x backup.sh

# 添加到 crontab (每天凌晨2点备份)
0 2 * * * /path/to/backup.sh
```

## 升级

### 使用 Docker Compose 升级

```bash
# 拉取最新镜像
docker-compose pull

# 重启服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 数据库迁移

升级时会自动运行数据库迁移。如需手动迁移：

```bash
docker-compose exec backend ./datamap migrate
```

## 监控

### 健康检查

```bash
# API 健康检查
curl http://localhost:8080/health

# 前端健康检查
curl http://localhost/health
```

### 日志查看

```bash
# 查看所有日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend
```

## 故障排查

### 服务无法启动

```bash
# 检查端口占用
netstat -tlnp | grep -E '8080|80|5432'

# 查看详细日志
docker-compose logs --tail=100

# 检查配置文件
docker-compose config
```

### 数据库连接失败

```bash
# 测试数据库连接
docker-compose exec postgres pg_isready -U datamap

# 重置数据库（会丢失数据）
make db-reset
```

## 卸载

```bash
# 停止服务并删除数据
make clean-all
# 或
docker-compose down -v --rmi local
```

## 环境变量说明

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_USER` | 数据库用户名 | datamap |
| `DB_PASSWORD` | 数据库密码 | datamap123 |
| `DB_NAME` | 数据库名称 | datamap |
| `ENCRYPTION_KEY` | Docker Compose 使用的 AES 加密密钥(32字节) | change-me... |
| `JWT_SECRET` | Docker Compose 使用的 JWT 签名密钥 | change-me... |
| `API_PORT` | API端口 | 8080 |
| `WEB_PORT` | Web端口 | 80 |
| `LOG_LEVEL` | 日志级别 | info |

首次启动会自动创建默认管理员账号 `admin / admin123`，上线后请立即修改或重建。

## 获取更多帮助

- GitHub Issues: https://github.com/yourusername/datamap-lite/issues
- 文档: https://datamap-lite.readthedocs.io
