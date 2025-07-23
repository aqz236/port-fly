# SSH 终端功能测试指南

## 功能概述

我们已经成功实现了 SSH 终端功能，包括：

✅ **完整的主机节点组件** - 显示主机信息、状态和操作按钮
✅ **SSH 连接 API** - 后端 SSH 连接、断开、测试和命令执行
✅ **WebSocket 终端** - 实时 SSH 终端界面
✅ **前后端通信** - Vite 代理配置和 API 集成
✅ **错误处理** - 连接测试和详细错误提示

## 当前状态

- 前端代理配置 ✅ 正常工作
- 后端 API ✅ 正常响应  
- SSH 连接 ❌ 需要有效的主机配置

## 测试步骤

### 1. 配置测试主机

为了测试 SSH 终端功能，你需要：

#### 选项 A: 使用本地主机（推荐用于测试）
```sql
-- 在数据库中添加本地主机配置
INSERT INTO hosts (name, hostname, port, username, auth_method, password, group_id, created_at, updated_at) 
VALUES ('本地测试', 'localhost', 22, 'your_username', 'password', 'your_password', 1, datetime('now'), datetime('now'));
```

#### 选项 B: 使用远程主机
确保远程主机：
- SSH 服务正在运行
- 端口是开放的
- 认证信息正确

### 2. 测试连接流程

1. **点击"连接主机"按钮**
   - 首先会进行连接测试
   - 测试成功后再进行实际连接
   - 查看浏览器控制台的详细日志

2. **连接成功后点击"终端"按钮**
   - 会打开 SSH 终端界面
   - WebSocket 连接到后端
   - 可以进行命令行操作

### 3. 故障排除

#### SSH 连接失败
```
错误: "SSH handshake failed: ssh: handshake failed: EOF"
```

**可能原因：**
- 主机地址不正确
- SSH 端口未开放
- 认证信息错误
- SSH 服务未运行

**解决方案：**
1. 检查主机配置
2. 验证 SSH 服务状态
3. 测试手动 SSH 连接: `ssh username@hostname -p port`

#### WebSocket 连接失败
```
错误: "WebSocket connection to 'ws://localhost:8080/ws/terminal/1' failed"
```

**可能原因：**
- 后端服务器未运行
- WebSocket 路由配置错误

**解决方案：**
1. 确保后端服务器运行在端口 8080
2. 检查 WebSocket 路由配置

## 架构说明

### 前端架构
```
HostNode (主机节点)
  ├── 连接按钮 → useCanvasHandlers.handleHostConnect
  ├── 终端按钮 → SSHTerminal 组件
  └── 状态显示 → 实时更新主机状态

SSHTerminal (SSH终端)
  ├── XTerm.js 终端界面
  ├── WebSocket 通信
  └── 实时命令交互
```

### 后端架构
```
API 端点:
  ├── POST /api/v1/hosts/:id/connect    # 连接主机
  ├── POST /api/v1/hosts/:id/disconnect # 断开连接
  ├── POST /api/v1/hosts/:id/test       # 测试连接
  └── POST /api/v1/hosts/:id/execute    # 执行命令

WebSocket:
  └── /ws/terminal/:hostId              # SSH 终端会话
```

## 下一步改进

1. **主机管理界面** - 可视化添加/编辑主机
2. **密钥认证** - 支持 SSH 密钥文件上传
3. **会话管理** - 多终端会话支持
4. **终端录制** - 会话记录和回放
5. **文件传输** - SFTP 文件管理
6. **端口转发** - 可视化端口转发管理

## 结论

SSH 终端功能的核心架构已经完成，包含了：
- 完整的前后端通信
- 实时 WebSocket 终端
- 错误处理和状态管理
- 可扩展的组件架构

只需要配置有效的主机连接信息即可开始使用。
