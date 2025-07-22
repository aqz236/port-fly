根据您的需求（基于Go的SSH端口转发工具 + Angular可视化界面 + 命令行），我将设计一个模块化、可扩展的目录结构，并列出关键开源库：

---

### **整体项目结构（单一仓库）**

```
port-fly/
├── core/                  # Go核心引擎（独立模块）
├── cli/                   # Go命令行接口
├── server/                # Go HTTP API服务
├── web-ui/                # Angular前端（独立项目）
└── configs/               # 配置文件模板
```

---

### **1. 核心引擎模块 (core/)**

**路径**:** **`core/`
**职责**: SSH连接管理、端口转发逻辑、安全处理**
文件结构**:

```
core/
├── ssh/
│   ├── client.go           # SSH客户端管理（连接池/状态）
│   ├── tunnel.go           # 端口转发核心逻辑（远程↔本地）
│   ├── auth.go             # 认证处理（密码/密钥/agent）
│   └── crypto_utils.go     # 密钥加解密工具
├── manager/
│   ├── session_manager.go  # 转发会话生命周期管理
│   └── rule_manager.go     # 转发规则存储/加载
├── models/
│   ├── session.go          # 会话数据结构
│   └── config.go           # 配置数据结构
└── utils/
    ├── logger.go           # 统一日志接口
    └── network_utils.go    # 网络工具函数
```

**关键依赖库**:

* SSH协议:** **`golang.org/x/crypto/ssh`
* 并发控制:** **`sync` (原生)
* 连接池:** **`github.com/fatih/pool` (可选)
* 配置文件:** **`github.com/spf13/viper`

---

### **2. 命令行模块 (cli/)**

**路径**:** **`cli/`
**职责**: 命令行交互、指令解析**
文件结构**:

```
cli/
├── cmd/
│   ├── root.go             # 根命令
│   ├── start.go            # 启动转发
│   ├── stop.go             # 停止会话
│   ├── list.go             # 查看活动会话
│   └── config.go           # 配置管理命令
└── console/
    ├── output.go           # 控制台输出格式化
    └── interactive.go      # 交互式向导
```

**关键依赖库**:

* CLI框架:** **`github.com/spf13/cobra`
* 用户输入:** **`github.com/AlecAivazis/survey/v2` (问卷调查式输入)
* 表格输出:** **`github.com/jedib0t/go-pretty/v6/table`

---

### **3. HTTP API 服务模块 (server/)**

**路径**:** **`server/`
**职责**: 提供REST API给Angular前端**
文件结构**:

```
server/
├── api/
│   ├── v1/
│   │   ├── session_handler.go  # 会话管理API
│   │   └── rule_handler.go     # 规则配置API
├── middleware/
│   ├── auth.go                # JWT认证
│   └── logger.go              # 请求日志
├── websocket/
│   └── log_stream.go          # 实时日志推送
├── storage/
│   └── boltdb_store.go        # BoltDB持久化存储
└── server.go                  # 服务启动入口
```

**关键依赖库**:

* Web框架:** **`github.com/gin-gonic/gin`
* 实时通信:** **`github.com/gorilla/websocket`
* 持久化存储:** **`go.etcd.io/bbolt`
* JWT认证:** **`github.com/golang-jwt/jwt/v5`

---

### **4. Angular前端模块 (web-ui/)**

```
web-ui/
├── src/
│   ├── app/
│   │   ├── core/              # 核心模块
│   │   ├── services/          # API服务
│   │   │   ├── api.service.ts 
│   │   │   └── ws.service.ts   # WebSocket服务
│   │   ├── pages/
│   │   │   ├── dashboard/      # 仪表盘
│   │   │   ├── session-mgmt/   # 会话管理
│   │   │   ├── rule-editor/    # 规则编辑器
│   │   │   └── logs/           # 实时日志
│   │   ├── components/
│   │   │   ├── session-card/
│   │   │   └── port-visualizer/ # 端口可视化组件
│   │   ├── models/             # TypeScript类型定义
│   │   └── assets/             # 静态资源
├── angular.json
└── package.json
```

**关键依赖库**:

* UI组件:** **`Angular Material`
* 图表:** **`ngx-echarts`
* WebSocket:** **`rxjs/webSocket`
* 表单处理:** **`ReactiveFormsModule`

---

### **5. 配置文件模板 (configs/)**

```
configs/
├── default.yaml         # 默认配置
└── bolt-schema.json     # BoltDB数据模型定义
```

---

### **关键技术栈总结**

| **模块** | **技术**         | **重要依赖库**                      |
| -------------- | ---------------------- | ----------------------------------------- |
| 核心引擎       | Go                     | golang.org/x/crypto/ssh, go.etcd.io/bbolt |
| 命令行接口     | Cobra框架              | github.com/spf13/cobra, go-pretty/table   |
| API服务        | Gin + BoltDB           | gin-gonic/gin, gorilla/websocket          |
| 前端可视化     | Angular 最新版         | Angular Material, ngx-echarts, RxJS       |
| 部署           | 单二进制 + Web静态资源 | 使用Go静态文件嵌入：`embed` 包          |

---

### **模块间交互流程**

```
flowchart LR
    A[Angular UI] -->|HTTP API| B(Go API Server)
    B -->|控制指令| C[Core Engine]
    C -->|状态通知| B
    B -->|WebSocket推送| A
    D[CLI] -->|直接调用| C
```

---

### **优势设计**

1. **分层解耦**
   * 核心引擎完全独立，可被CLI/API复用
   * API服务与核心通过接口交互（`SessionManager` interface）
2. **扩展性**
   * 新增认证方式：在 `core/ssh/auth.go`中实现新Provider
   * 支持新存储：在 `server/storage/`添加新适配器
3. **安全增强**
   * JWT认证 + HTTPS支持
   * 敏感配置加密存储（使用 `crypto_utils`）
4. **一键构建**
   ```
   # 编译带前端资源的二进制
   cd server && go build -tags embed

   # 独立构建前端
   cd web-ui && ng build --configuration production
   ```

此结构支持：独立开发核心引擎、CLI与Web可并行开发、前端可单独部署。如果需要容器化部署，可新增 `Dockerfile`支持多阶段构建。
