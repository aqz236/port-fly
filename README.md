# PortFly - SSH隧道管理器

PortFly是一个现代化的SSH隧道管理工具，支持Termius风格的主机和端口分组管理。

## 项目结构

```
port-fly/
├── core/                    # 核心功能模块
│   ├── models/              # 数据模型
│   │   ├── ssh.go           # SSH配置模型
│   │   └── groups.go        # 主机/端口分组模型
│   ├── ssh/                 # SSH核心功能
│   │   ├── client.go        # SSH客户端和连接池
│   │   ├── auth.go          # 多种认证方式
│   │   ├── tunnel.go        # 端口转发实现
│   │   └── crypto_utils.go  # 加密工具
│   ├── utils/               # 工具模块
│   │   ├── logger.go        # 结构化日志
│   │   └── network.go       # 网络工具
│   └── manager/             # 会话管理
│       └── session.go       # 隧道会话管理
├── cli/                     # 命令行工具
│   └── cmd/         
│       ├── root.go          # 根命令
│       └── start.go         # 启动隧道命令
├── server/                  # HTTP API服务器
│   ├── storage/             # 存储层
│   │   ├── interface.go     # 存储接口
│   │   ├── sqlite.go        # SQLite实现
│   │   ├── postgres.go      # PostgreSQL实现
│   │   ├── mysql.go         # MySQL实现
│   │   └── factory.go       # 存储工厂
│   ├── handlers/            # HTTP处理器
│   │   └── handlers.go      # API端点实现
│   ├── middleware/          # 中间件
│   │   └── middleware.go    # 请求日志等中间件
│   └── server.go            # 服务器主程序
├── cmd/                     # 可执行程序入口
│   ├── cli/                 # CLI入口
│   │   └── main.go
│   └── server/              # 服务器入口
│       └── main.go
└── docs/                    # 文档
    └── PortFly设计文档.md
```

## 核心特性

### 🚀 已完成功能

#### 1. SSH核心引擎

- ✅ **多种认证方式**: 密码、私钥、SSH代理
- ✅ **连接池管理**: 复用SSH连接，提升性能
- ✅ **端口转发**: 支持本地(-L)、远程(-R)、动态(-D)转发
- ✅ **连接保活**: 自动维持SSH连接稳定性
- ✅ **错误重试**: 智能重连机制

#### 2. 命令行工具

- ✅ **Cobra框架**: 现代化CLI界面
- ✅ **交互式认证**: 安全的密码输入
- ✅ **实时状态**: 连接状态和流量监控
- ✅ **配置管理**: 支持配置文件和命令行参数

#### 3. 数据模型 (Termius风格)

- ✅ **主机分组**: 按环境/项目组织主机
- ✅ **端口分组**: 按服务类型组织端口转发
- ✅ **关联管理**: 主机和端口转发的关联关系
- ✅ **扩展属性**: 颜色、图标、标签、元数据支持

#### 4. 存储层 (多数据库支持)

- ✅ **SQLite**: 默认嵌入式数据库
- ✅ **PostgreSQL**: 生产环境数据库
- ✅ **MySQL**: 企业级数据库支持
- ✅ **GORM集成**: 自动迁移和ORM功能

#### 5. HTTP API服务器

- ✅ **RESTful API**: 完整的CRUD操作
- ✅ **Gin框架**: 高性能HTTP服务器
- ✅ **中间件支持**: CORS、日志、错误处理
- ✅ **WebSocket就绪**: 实时通信准备
- ✅ **统计查询**: 分组统计和搜索功能

## 技术栈

### 后端技术

- **Go 1.21+**: 主要编程语言
- **golang.org/x/crypto/ssh**: SSH客户端库
- **Cobra**: CLI框架
- **Gin**: HTTP Web框架
- **GORM**: ORM框架
- **SQLite/PostgreSQL/MySQL**: 数据库支持

### 数据库设计

```sql
-- 主机分组
host_groups (id, name, description, color, icon, tags, metadata, timestamps)

-- 主机配置
hosts (id, name, hostname, port, username, auth_method, host_group_id, timestamps)

-- 端口分组  
port_groups (id, name, description, color, auto_start, max_concurrent, timestamps)

-- 端口转发配置
port_forwards (id, name, type, local_port, remote_host, remote_port, host_id, port_group_id, timestamps)

-- 隧道会话
tunnel_sessions (id, status, start_time, end_time, error_message, host_id, port_forward_id, timestamps)
```

## 快速开始

### 1. 构建项目

```bash
# 构建CLI工具
go build -o bin/portfly ./cmd/cli

# 构建API服务器
go build -o bin/portfly-server ./cmd/server
```

### 2. 使用CLI工具

```bash
# 启动本地端口转发 (SSH -L)
./bin/portfly start -L 8080:localhost:80 root@47.236.206.128:8355

# 启动远程端口转发 (SSH -R)  
./bin/portfly start -R 9000:localhost:3000 user@remote-server.com

# 启动SOCKS5代理 (SSH -D)
./bin/portfly start -D 1080 user@proxy-server.com
```

### 3. 启动API服务器

```bash
# 使用默认配置启动 (SQLite数据库)
./bin/portfly-server

# 服务器将在 http://localhost:8080 启动
```

### 4. API使用示例

```bash
# 健康检查
curl http://localhost:8080/health

# 创建主机分组
curl -X POST http://localhost:8080/api/v1/host-groups \
  -H "Content-Type: application/json" \
  -d '{"name": "Production", "description": "生产环境", "color": "#ff6b6b"}'

# 创建主机配置
curl -X POST http://localhost:8080/api/v1/hosts \
  -H "Content-Type: application/json" \
  -d '{"name": "Web Server", "hostname": "server.com", "port": 22, "username": "deploy", "host_group_id": 1}'

# 获取所有主机
curl http://localhost:8080/api/v1/hosts

# 搜索主机
curl "http://localhost:8080/api/v1/hosts/search?q=web"
```

## API端点

### 主机分组管理

- `GET /api/v1/host-groups` - 获取所有主机分组
- `POST /api/v1/host-groups` - 创建主机分组
- `GET /api/v1/host-groups/:id` - 获取特定主机分组
- `PUT /api/v1/host-groups/:id` - 更新主机分组
- `DELETE /api/v1/host-groups/:id` - 删除主机分组
- `GET /api/v1/host-groups/:id/stats` - 获取分组统计

### 主机管理

- `GET /api/v1/hosts` - 获取所有主机
- `POST /api/v1/hosts` - 创建主机
- `GET /api/v1/hosts/:id` - 获取特定主机
- `PUT /api/v1/hosts/:id` - 更新主机
- `DELETE /api/v1/hosts/:id` - 删除主机
- `GET /api/v1/hosts/search?q=query` - 搜索主机

### 端口分组管理

- `GET /api/v1/port-groups` - 获取所有端口分组
- `POST /api/v1/port-groups` - 创建端口分组
- `GET /api/v1/port-groups/:id` - 获取特定端口分组
- `PUT /api/v1/port-groups/:id` - 更新端口分组
- `DELETE /api/v1/port-groups/:id` - 删除端口分组

### 端口转发管理

- `GET /api/v1/port-forwards` - 获取所有端口转发
- `POST /api/v1/port-forwards` - 创建端口转发
- `GET /api/v1/port-forwards/:id` - 获取特定端口转发
- `PUT /api/v1/port-forwards/:id` - 更新端口转发
- `DELETE /api/v1/port-forwards/:id` - 删除端口转发

### 隧道会话管理

- `GET /api/v1/sessions` - 获取所有会话
- `POST /api/v1/sessions` - 创建会话
- `GET /api/v1/sessions/active` - 获取活跃会话
- `POST /api/v1/sessions/:id/start` - 启动隧道
- `POST /api/v1/sessions/:id/stop` - 停止隧道

## 测试验证

我们已经成功测试了以下功能：

### ✅ CLI工具测试

- SSH连接和认证 (密码方式)
- 本地端口转发 (-L 8080:localhost:80)
- 实际服务器连接 (47.236.206.128:8355)

### ✅ API服务器测试

- 数据库自动创建和迁移
- RESTful API端点正常工作
- 主机分组CRUD操作
- 主机配置CRUD操作
- 端口分组和端口转发管理
- 搜索和统计功能
- SQLite数据持久化

### ✅ 数据库测试

- GORM自动迁移正常
- 关联查询正常工作
- 统计查询正确
- 搜索功能正常

## 下一步计划

### 🚧 开发中功能

- [ ] **前端界面**: Remix + Tailwind CSS
- [ ] **隧道控制**: API启动/停止隧道
- [ ] **实时监控**: WebSocket状态更新
- [ ] **会话持久化**: 重启后恢复隧道

### 🔮 未来功能

- [ ] **用户认证**: JWT登录系统
- [ ] **配置同步**: 云端配置同步
- [ ] **监控仪表板**: 流量和性能监控
- [ ] **批量操作**: 批量管理隧道

## 项目亮点

1. **模块化设计**: 清晰的分层架构，易于维护和扩展
2. **生产就绪**: 完整的错误处理、日志记录、数据验证
3. **Termius风格**: 现代化的分组管理概念
4. **多数据库支持**: SQLite、PostgreSQL、MySQL
5. **高性能**: 连接池、并发处理、资源复用
6. **实际验证**: 真实服务器测试验证功能

这个项目展示了Go语言在系统编程中的强大能力，结合了网络编程、数据库操作、Web开发等多个技术领域。
