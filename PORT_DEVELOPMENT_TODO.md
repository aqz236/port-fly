# Port 功能开发 Todo List

## 项目背景
开发一个独立的Port模型系统，支持：
- Port可以独立创建和管理，分为Remote_Port和Local_Port两种类型
- Port有状态管理（可用/不可用）
- Host连线Port时自动测试端口状态是否可用
- Remote_Port连线Local_Port时实现端口转发
- 前端ReactFlow中可视化显示Port节点
- 实时状态检测和更新

## 工作流程设计

### 连线逻辑
1. **Host → Port连线**：
   - 自动测试Host到Port的连通性
   - 更新Port状态（available/unavailable）
   - 不创建端口转发，仅用于状态检测

2. **Remote_Port → Local_Port连线**：
   - 创建实际的端口转发隧道
   - Remote_Port作为远程端口（源）
   - Local_Port作为本地端口（目标）
   - 建立SSH隧道实现端口转发

### 节点类型
- **Host节点**：SSH连接主机
- **Remote_Port节点**：远程端口（需要转发的源端口）
- **Local_Port节点**：本地端口（转发的目标端口）

### 完整流程
```
Host → Remote_Port → Local_Port
 ↓        ↓           ↓
连通性测试  状态检测    端口转发
```

## 开发任务清单

### 🗄️ 数据层（Backend - Models & Storage）

#### 1. 创建Port模型
- [x] **创建 `core/models/port.go`** ✅
  - Port结构体定义（独立于PortForward）
  - 状态枚举：`available`, `unavailable`, `active`, `error`, `connecting`
  - 类型枚举：`remote_port`, `local_port`
  - 关联关系：Group, Host（可选）
  - 元数据字段：名称、描述、端口号、绑定地址等
  - JSON序列化标签

#### 2. 扩展Storage接口
- [x] **更新 `server/storage/interface.go`** ✅
  - 添加Port相关CRUD方法
  - `CreatePort`, `GetPort`, `GetPorts`, `UpdatePort`, `DeletePort`
  - `GetPortsByGroup`, `GetPortsByHost`
  - `GetPortStats`, `SearchPorts`
  - `UpdatePortStatus` - 状态更新方法

#### 3. 实现SQLite存储
- [x] **创建 `server/storage/sqlite/port_operations.go`** ✅
  - 实现所有Port相关的数据库操作
  - 数据库迁移脚本
  - 索引优化
  - 状态查询优化

### 🌐 API层（Backend - Handlers & Routes）

#### 4. 创建Port Handlers
- [x] **创建 `server/handlers/port.handlers.go`** ✅
  - GetPorts - 获取端口列表（支持按Group/Host过滤）
  - CreatePort - 创建新端口
  - GetPort - 获取单个端口详情
  - UpdatePort - 更新端口信息
  - DeletePort - 删除端口
  - GetPortStats - 获取端口统计信息

#### 5. 端口控制handlers
- [x] **在 `port.handlers.go` 中添加控制方法** ✅
  - TestPortConnection - 测试Host到Port的连通性
  - CreatePortForward - Remote_Port连线Local_Port时创建端口转发
  - RemovePortForward - 断开连线时移除端口转发
  - UpdatePortStatus - 更新端口状态

#### 6. 添加API路由
- [x] **更新 `server/server.go`** ✅
  - 添加 `/api/v1/ports` 路由组
  - 注册所有Port相关的endpoints
  - 支持RESTful API设计

### 🔧 核心功能（Backend - Core Logic）

#### 7. 端口管理器
- [ ] **创建 `core/manager/port_manager.go`**
  - Port生命周期管理
  - Host到Port的连通性测试
  - Remote_Port到Local_Port的端口转发管理
  - 状态监控和更新
  - 连接健康检查

#### 8. 端口状态监控
- [ ] **扩展现有监控系统**
  - 定期检查Host到Port的连通性
  - 监控Remote_Port到Local_Port的转发状态
  - 状态变化通知（WebSocket）
  - 统计信息收集

### 🎨 前端API客户端

#### 9. 更新类型定义
- [x] **完善 `web-ui/app/shared/types/port/index.ts`** ✅
  - 确保类型定义与后端模型一致
  - 添加状态转换枚举
  - 添加控制操作类型

#### 10. 完善API客户端
- [x] **完善 `web-ui/app/shared/api/port.ts`** ✅
  - 实现所有CRUD操作
  - 添加连通性测试方法（Host->Port）
  - 添加端口转发创建/删除方法（Remote_Port->Local_Port）
  - 添加状态查询方法
  - 错误处理

#### 11. React Query Hooks
- [ ] **创建 `web-ui/app/shared/api/hooks/use-ports.ts`**
  - usePort, usePorts
  - useCreatePort, useUpdatePort, useDeletePort
  - useTestPortConnection (Host->Port连通性测试)
  - useCreatePortForward (Remote_Port->Local_Port转发)
  - usePortStats
  - 实时状态更新hooks

### 🎯 ReactFlow组件

#### 12. Port节点组件
- [ ] **创建 `web-ui/app/features/ports/components/PortNode.tsx`**
  - 区分Remote_Port和Local_Port的可视化显示
  - 支持不同类型的连线操作
  - 状态实时更新
  - Host->Port连线时自动测试连通性
  - Remote_Port->Local_Port连线时创建端口转发

#### 13. Port管理界面
- [ ] **创建Port管理组件**
  - `web-ui/app/features/ports/components/PortList.tsx`
  - `web-ui/app/features/ports/components/CreatePortModal.tsx`
  - `web-ui/app/features/ports/components/EditPortModal.tsx`
  - `web-ui/app/features/ports/containers/PortManager.tsx`

#### 14. 画布集成
- [ ] **更新ReactFlow画布**
  - 添加Remote_Port和Local_Port节点类型
  - 实现Host->Port连线逻辑（连通性测试）
  - 实现Remote_Port->Local_Port连线逻辑（端口转发）
  - 连线创建时的自动化处理
  - 状态同步和更新

### 🔄 实时功能

#### 15. WebSocket状态更新
- [ ] **扩展WebSocket系统**
  - Port状态变化推送
  - 连接状态实时更新
  - 统计信息推送

#### 16. 健康检查
- [ ] **实现端口健康检查**
  - Host到Port的连通性检查
  - Remote_Port到Local_Port转发状态检查
  - 自动重连机制

### 🧪 测试

#### 17. 后端测试
- [ ] **创建单元测试**
  - Port模型测试
  - Storage层测试
  - Handler测试
  - Manager测试

#### 18. 前端测试
- [ ] **创建组件测试**
  - Port组件测试
  - Hook测试
  - 集成测试

### 📚 文档

#### 19. API文档
- [ ] **创建API文档**
  - Port相关API endpoints
  - 请求/响应示例
  - 状态码说明

#### 20. 使用说明
- [ ] **创建用户指南**
  - Port创建和管理
  - 连线操作说明
  - 故障排除

## 开发优先级

### Phase 1: 核心功能（高优先级）
1. Port模型和数据库存储
2. 基础CRUD API
3. 简单的Port管理界面

### Phase 2: 状态管理（中优先级）
4. 端口状态监控
5. WebSocket实时更新
6. ReactFlow节点集成

### Phase 3: 高级功能（低优先级）
7. 健康检查和自动重连
8. 统计信息和监控
9. 完整的测试覆盖

## 技术要求

### 后端
- Go 1.19+
- Gin框架
- GORM ORM
- SQLite数据库
- WebSocket支持

### 前端
- React 18+
- TypeScript
- Remix框架
- ReactFlow
- Tanstack Query
- WebSocket客户端

## 预期交付物

1. **独立的Port模型系统** - 可以独立创建、管理Remote_Port和Local_Port
2. **连通性测试系统** - Host连线Port时自动测试状态
3. **端口转发系统** - Remote_Port连线Local_Port时创建端口转发
4. **ReactFlow集成** - 可视化Port节点和连线操作
5. **实时状态更新** - WebSocket推送状态变化
6. **完整的CRUD操作** - 创建、读取、更新、删除Port

## 验收标准

- [ ] 可以独立创建Remote_Port和Local_Port，不依赖Host
- [ ] Host连线Port时能够自动测试连通性并更新状态
- [ ] Remote_Port连线Local_Port时能够创建端口转发
- [ ] ReactFlow中可以正常显示不同类型的Port节点
- [ ] 端口状态能够实时更新显示
- [ ] 前端界面响应流畅，状态同步及时
- [ ] API接口完整，错误处理合理
- [ ] 代码质量良好，测试覆盖充分

---

*本开发计划基于现有项目架构设计，确保与现有Host、Group、Project等模型的良好集成。*
