# PortFly - SSH隧道管理器

PortFly是一个现代化的SSH隧道管理工具，采用灵活的三层架构设计：**Project（项目）-> Group（组）-> Resource（主机+端口）**。

## 🏗️ 架构设计

### 数据模型架构

```
Project (项目/工作空间)
    ├── Group (资源组)
    │   ├── Host (主机)
    │   └── PortForward (端口转发)
    └── Group (资源组)
        ├── Host (主机)
        └── PortForward (端口转发)
```

### 核心特点

- **项目级管理**：按项目或环境隔离资源
- **灵活分组**：每个组可包含主机和端口转发规则
- **统一管理**：主机和端口在同一组内便于关联管理
- **多数据库支持**：SQLite、PostgreSQL、MySQL
- **现代化API**：RESTful API + WebSocket实时通信

## 📂 项目结构

```txt
port-fly/
├── core/                    # 核心功能模块
│   ├── models/              # 数据模型
│   │   ├── project.go       # 项目、组、主机、端口模型
│   │   ├── session.go       # 隧道会话模型
│   │   └── config.go        # 配置模型
│   ├── ssh/                 # SSH核心功能
│   │   ├── client.go        # SSH客户端和连接池
│   │   ├── auth.go          # 多种认证方式支持
│   │   ├── tunnel.go        # 端口转发实现
│   │   └── crypto_utils.go  # 加密工具
│   ├── utils/               # 工具模块
│   │   ├── logger.go        # 结构化日志
│   │   └── network_utils.go # 网络工具
│   └── manager/             # 会话管理
│       └── session_manager.go # 隧道会话管理器
├── cli/                     # 命令行工具
│   ├── cmd/                 # 命令定义
│   │   ├── root.go          # 根命令
│   │   └── start.go         # 启动隧道命令
│   └── console/             # 交互式控制台
├── server/                  # HTTP API服务器
│   ├── storage/             # 存储抽象层
│   │   ├── interface.go     # 存储接口定义
│   │   ├── sqlite.go        # SQLite实现
│   │   ├── postgres.go      # PostgreSQL实现
│   │   └── mysql.go         # MySQL实现
│   ├── handlers/            # HTTP请求处理器
│   │   └── handlers.go      # API端点实现
│   ├── middleware/          # 中间件
│   │   └── middleware.go    # CORS、日志等中间件
│   ├── api/v1/              # API版本管理
│   └── server.go            # 服务器主程序
├── web-ui/                  # 前端界面
│   ├── app/                 # Remix应用
│   │   ├── components/      # React组件
│   │   ├── routes/          # 页面路由
│   │   ├── lib/             # 工具库
│   │   └── types/           # TypeScript类型定义
│   └── public/              # 静态资源
├── cmd/                     # 可执行程序入口
│   ├── cli/main.go          # CLI入口
│   └── server/main.go       # 服务器入口
└── docs/                    # 文档
```

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

#### 3. 数据模型

- ✅ **主机分组**: 按环境/项目组织主机
- ✅ **端口分组**: 按服务类型组织端口转发
- ✅ **关联管理**: 主机和端口转发的关联关系
- ✅ **扩展属性**: 颜色、图标、标签、元数据支持

#### 4. 存储层 (多数据库支持)

- ✅ **SQLite**: 默认嵌入式数据库
- ✅ **PostgreSQL**: 生产环境数据库
- ✅ **MySQL**: 高性能数据库支持
- ✅ **GORM集成**: 自动迁移和ORM功能
- ✅ **连接池**: 数据库连接池优化

### 4. REST API服务

- ✅ **RESTful设计**: 完整的CRUD操作
- ✅ **Gin框架**: 高性能HTTP服务器
- ✅ **中间件支持**: CORS、日志、认证、错误处理
- ✅ **WebSocket**: 实时状态更新
- ✅ **统计分析**: 项目/组/资源统计

### 5. 命令行工具

- ✅ **Cobra框架**: 现代化CLI界面
- ✅ **交互式操作**: 安全的密码输入
- ✅ **实时监控**: 连接状态和流量统计
- ✅ **配置管理**: 多种配置方式

### 6. Web管理界面

- ✅ **React + Remix**: 现代化前端框架
- ✅ **TypeScript**: 类型安全
- ✅ **Tailwind CSS**: 现代UI设计
- ✅ **实时更新**: WebSocket集成

## 📊 数据库设计

### 核心表结构

```sql
-- 项目表
projects (
    id, name, description, color, icon, 
    is_default, metadata, created_at, updated_at
)

-- 资源组表
groups (
    id, name, description, color, icon, 
    project_id, tags, metadata, created_at, updated_at
)

-- 主机表  
hosts (
    id, name, hostname, port, username, auth_method,
    group_id, status, connection_count, tags, metadata,
    created_at, updated_at
)

-- 端口转发表
port_forwards (
    id, name, type, local_port, remote_host, remote_port,
    group_id, host_id, auto_start, status, tags, metadata,
    created_at, updated_at
)

-- 隧道会话表
tunnel_sessions (
    id, status, start_time, end_time, error_message,
    host_id, port_forward_id, data_transferred, pid,
    created_at, updated_at
)
```

## 🛠️ 技术栈

### 后端技术

- **Go 1.21+**: 主要编程语言
- **golang.org/x/crypto/ssh**: SSH客户端库
- **Cobra**: 命令行框架
- **Gin**: HTTP Web框架
- **GORM**: ORM框架
- **SQLite/PostgreSQL/MySQL**: 数据库支持

### 前端技术

- **React 18**: 前端框架
- **Remix**: 全栈Web框架
- **TypeScript**: 类型安全
- **Tailwind CSS**: CSS框架
- **Vite**: 构建工具

## 🚀 快速开始

### 1. 环境要求

- Go 1.21+
- Bun.js (前端开发)
- SQLite/PostgreSQL/MySQL (任选其一)

### 2. 构建项目

```bash
# 克隆项目
git clone https://github.com/aqz236/port-fly.git
cd port-fly

# 构建后端
go mod download
# 构建前端
cd web-ui
bun install
bun run build
cd ..
```

### 3. 启动服务

```bash
# 启动API服务器
./bin/portfly-server

# 或使用不同数据库
PORTFLY_DB_TYPE=postgres ./bin/portfly-server
PORTFLY_DB_TYPE=mysql ./bin/portfly-server
```

### 4. 使用CLI工具

```bash
# 创建项目
./bin/portfly-cli --help
```

## 📚 API文档

### 核心端点

#### 项目管理

```http
GET    /api/v1/projects          # 获取所有项目
POST   /api/v1/projects          # 创建项目
GET    /api/v1/projects/:id      # 获取单个项目
PUT    /api/v1/projects/:id      # 更新项目
DELETE /api/v1/projects/:id      # 删除项目
GET    /api/v1/projects/:id/stats # 获取项目统计
```

#### 组管理

```http
GET    /api/v1/groups            # 获取所有组
POST   /api/v1/groups            # 创建组
GET    /api/v1/groups/:id        # 获取单个组
PUT    /api/v1/groups/:id        # 更新组
DELETE /api/v1/groups/:id        # 删除组
GET    /api/v1/groups/:id/stats  # 获取组统计
```

#### 主机管理

```http
GET    /api/v1/hosts             # 获取所有主机
POST   /api/v1/hosts             # 创建主机
GET    /api/v1/hosts/:id         # 获取单个主机
PUT    /api/v1/hosts/:id         # 更新主机
DELETE /api/v1/hosts/:id         # 删除主机
GET    /api/v1/hosts/search      # 搜索主机
```

#### 端口转发管理

```http
GET    /api/v1/port-forwards     # 获取所有端口转发
POST   /api/v1/port-forwards     # 创建端口转发
GET    /api/v1/port-forwards/:id # 获取单个端口转发
PUT    /api/v1/port-forwards/:id # 更新端口转发
DELETE /api/v1/port-forwards/:id # 删除端口转发
```

#### 隧道会话

```http
GET    /api/v1/sessions          # 获取所有会话
POST   /api/v1/sessions          # 创建会话
GET    /api/v1/sessions/active   # 获取活跃会话
POST   /api/v1/sessions/:id/start # 启动隧道
POST   /api/v1/sessions/:id/stop  # 停止隧道
```

## 🔧 配置说明

### 服务器配置

```yaml
# configs/default.yaml
server:
  host: "localhost"
  port: 8080
  mode: "release"
  enable_cors: true
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:5173"

database:
  type: "sqlite"        # sqlite, postgres, mysql
  host: "localhost"
  port: 5432
  database: "portfly"
  username: "portfly"
  password: "password"
  ssl_mode: "disable"
  options:
    max_open_conns: "25"
    max_idle_conns: "10"
    log_level: "info"
```

### 环境变量

```bash
# 数据库配置
export PORTFLY_DB_TYPE=postgres
export PORTFLY_DB_HOST=localhost
export PORTFLY_DB_PORT=5432
export PORTFLY_DB_NAME=portfly
export PORTFLY_DB_USER=portfly
export PORTFLY_DB_PASSWORD=password

# 服务器配置
export PORTFLY_SERVER_PORT=8080
export PORTFLY_SERVER_HOST=0.0.0.0
```

## 🧪 测试

```bash
# 运行单元测试
go test ./...

# 运行集成测试
go test -tags=integration ./...

# 测试覆盖率
go test -cover ./...
```

## 🤝 贡献

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🎯 路线图

- [ ] **Web界面完善**: 完整的前端管理界面
- [ ] **批量操作**: 支持批量管理主机和端口
- [ ] **监控面板**: 实时连接状态和流量监控
- [ ] **配置同步**: 支持配置文件导入/导出
- [ ] **用户认证**: 多用户支持和权限管理
- [ ] **插件系统**: 支持自定义扩展
- [ ] **Docker支持**: 容器化部署
- [ ] **集群模式**: 分布式部署支持

---

**PortFly** - 让SSH隧道管理变得简单高效！ 🚀
