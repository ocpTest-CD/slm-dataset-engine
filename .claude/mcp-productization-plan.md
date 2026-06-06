# MCP 产品化开发计划

- 日期：2026-06-07
- 状态：待执行
- 范围：将 MCP 能力从协议工具升级为可用产品工作台。
- 商业化：暂不做计费、套餐、发票、额度售卖。
- 权限：先做最简单的用户与角色体系，满足基础访问控制和审计。

## 产品目标

MCP 不是产品本身，只是 AI 调用工具、资源和提示词的协议入口。产品化的目标是把 MCP 能力封装成一个用户可登录、可管理、可观察、可审核、可下载产物的工作台。

第一阶段产品目标：

```text
登录
-> 创建工作空间和项目
-> 注册 MCP Server
-> 查看 Tools / Resources / Prompts
-> 在页面测试 Tool 调用
-> 生成 Invocation 和 Job
-> Worker 执行任务
-> 页面展示进度和结果
-> 生成 Artifact
-> 下载产物
-> 查看调用与操作日志
```

## 产品边界

### 当前必须做

- Web 工作台。
- MCP Server 注册与管理。
- Tool Schema 展示。
- Tool 调用测试。
- Tool Invocation 记录。
- Job 状态机与 Worker 心跳。
- Artifact 产物管理和下载。
- 最简单权限体系。
- 操作日志和调用审计。
- 数据源、任务、产物的项目级归属。

### 当前不做

- 商业化计费。
- 复杂多租户套餐。
- 插件市场。
- 高级权限策略引擎。
- 复杂流程画布。
- 大规模组织管理。
- 自定义计费报表。

## 简单权限体系

MVP 权限只保留三个角色：

```text
owner
member
viewer
```

### 角色能力

| 角色 | 能力 |
| --- | --- |
| owner | 管理工作空间、成员、项目、MCP Server、API Token、删除资源 |
| member | 创建项目、运行工具、上传数据、审核样本、导出产物 |
| viewer | 只读查看项目、调用记录、任务状态和产物 |

### 权限规则

- 每个用户必须属于至少一个 workspace。
- 每个 project、mcp_server、artifact、job 都归属 workspace。
- API 请求必须带 workspace 上下文。
- 所有写操作至少需要 member。
- 删除、成员管理和 token 管理需要 owner。
- viewer 不能触发工具调用、不能上传、不能导出、不能编辑样本。

### MVP 鉴权方式

第一版可以使用静态开发登录或简单 token：

```text
Authorization: Bearer <api_token>
```

Token 关联：

```text
user_id
workspace_id
role
expires_at
```

后续再升级为正式登录、OAuth、SSO 或 magic link。

## 总体架构

```text
React + Vite 工作台
        |
        v
Go API 控制面
        |
        +-- Auth / Workspace / Project
        +-- MCP Registry
        +-- Tool Invocation Gateway
        +-- Job Orchestrator
        +-- Artifact Manager
        +-- Audit Log
        |
        v
PostgreSQL
        |
        +-- users / workspaces / members
        +-- mcp_servers / mcp_tools / mcp_resources / mcp_prompts
        +-- tool_invocations / jobs / job_events
        +-- artifacts / artifact_files
        +-- audit_logs
        |
        v
Python / Go Worker
        |
        +-- MCP 调用代理
        +-- 数据处理
        +-- RAG 导出
        +-- 文件生成
        |
        v
本地 artifacts 或 Cloudflare R2
```

## 核心模块

### 1. Workspace 与用户

职责：

- 工作空间创建。
- 成员与角色。
- 简单 token 鉴权。
- 当前 workspace 上下文。

核心实体：

```text
users
workspaces
workspace_members
api_tokens
```

### 2. Project 工作台

职责：

- 管理项目。
- 聚合 MCP Server、数据源、任务、产物和日志。
- 承载用户实际工作流。

核心实体：

```text
projects
project_settings
```

### 3. MCP Registry

职责：

- 注册 MCP Server。
- 同步或手动录入 Tools、Resources、Prompts。
- 展示 Tool Schema。
- 启用或禁用工具。

核心实体：

```text
mcp_servers
mcp_tools
mcp_resources
mcp_prompts
```

### 4. Tool Invocation

职责：

- 记录每次工具调用。
- 记录输入、输出、状态、耗时、错误。
- 将长任务转为 Job。
- 支持页面测试调用和 AI 调用。

核心实体：

```text
tool_invocations
```

状态机：

```text
created -> authorized -> queued -> running -> succeeded
                                  |-> failed
                                  |-> canceled
```

### 5. Job 与 Worker

职责：

- 执行长任务。
- 记录 claim、running、heartbeat、complete、fail。
- 将 Worker 状态返回页面。
- 记录阶段进度和错误。

核心实体：

```text
jobs
job_events
```

状态机：

```text
pending -> claimed -> running -> succeeded
                     |-> failed -> pending
                     |-> failed_final
pending/running -> canceled
claimed/running -> stale -> pending
```

### 6. Artifact 产物管理

职责：

- 保存工具调用和任务生成的结果。
- 支持预览、下载、版本、manifest。
- 支持 RAG 数据集包、JSONL、ZIP、报告等格式。

核心实体：

```text
artifacts
artifact_files
```

状态机：

```text
draft -> building -> ready -> published
                  |-> failed
published -> deprecated
```

### 7. 审计日志

职责：

- 记录用户操作。
- 记录工具调用。
- 记录数据访问。
- 记录产物下载。

核心实体：

```text
audit_logs
```

## 数据库表规划

### 用户与权限

```text
users
- id
- email
- name
- created_at
- updated_at

workspaces
- id
- name
- created_at
- updated_at

workspace_members
- id
- workspace_id
- user_id
- role
- created_at

api_tokens
- id
- workspace_id
- user_id
- token_hash
- role
- expires_at
- created_at
- revoked_at
```

### MCP Registry

```text
mcp_servers
- id
- workspace_id
- project_id
- name
- endpoint
- transport
- status
- config
- created_at
- updated_at

mcp_tools
- id
- workspace_id
- server_id
- name
- description
- input_schema
- output_schema
- enabled
- created_at
- updated_at

mcp_resources
- id
- workspace_id
- server_id
- uri
- name
- description
- mime_type
- enabled
- created_at
- updated_at

mcp_prompts
- id
- workspace_id
- server_id
- name
- description
- arguments_schema
- enabled
- created_at
- updated_at
```

### 调用、任务和产物

```text
tool_invocations
- id
- workspace_id
- project_id
- server_id
- tool_id
- user_id
- status
- input
- output
- error_message
- duration_ms
- job_id
- created_at
- updated_at

jobs
- id
- workspace_id
- project_id
- invocation_id
- job_type
- status
- stage
- progress
- message
- payload
- result
- attempts
- max_attempts
- claimed_by
- heartbeat_at
- started_at
- finished_at
- error_message
- created_at
- updated_at

job_events
- id
- workspace_id
- job_id
- event_type
- stage
- progress
- message
- metadata
- created_at

artifacts
- id
- workspace_id
- project_id
- invocation_id
- job_id
- name
- artifact_type
- status
- manifest
- created_at
- updated_at

artifact_files
- id
- workspace_id
- artifact_id
- file_name
- file_path
- mime_type
- byte_size
- sha256
- created_at
```

### 审计

```text
audit_logs
- id
- workspace_id
- project_id
- user_id
- action
- target_type
- target_id
- metadata
- created_at
```

## API 规划

### Auth / Workspace

```text
GET  /api/me
GET  /api/workspaces
POST /api/workspaces
GET  /api/workspaces/:id/members
POST /api/workspaces/:id/tokens
DELETE /api/tokens/:id
```

### MCP Registry

```text
GET  /api/projects/:project_id/mcp-servers
POST /api/projects/:project_id/mcp-servers
GET  /api/mcp-servers/:id
PATCH /api/mcp-servers/:id
POST /api/mcp-servers/:id/sync

GET  /api/mcp-servers/:id/tools
GET  /api/mcp-tools/:id
PATCH /api/mcp-tools/:id
```

### Tool Invocation

```text
POST /api/mcp-tools/:id/invoke
GET  /api/projects/:project_id/invocations
GET  /api/tool-invocations/:id
```

### Jobs

```text
GET   /api/projects/:project_id/jobs
GET   /api/jobs/:id
POST  /api/jobs/claim
PATCH /api/jobs/:id/running
PATCH /api/jobs/:id/heartbeat
PATCH /api/jobs/:id/progress
PATCH /api/jobs/:id/complete
PATCH /api/jobs/:id/fail
GET   /api/jobs/:id/events
```

### Artifacts

```text
GET  /api/projects/:project_id/artifacts
GET  /api/artifacts/:id
GET  /api/artifacts/:id/files
GET  /api/artifact-files/:id/download
GET  /api/artifacts/:id/download
```

### Audit

```text
GET /api/projects/:project_id/audit-logs
GET /api/workspaces/:workspace_id/audit-logs
```

## 前端页面规划

```text
/dashboard
  总览、最近调用、失败任务、最近产物

/projects
  项目列表

/projects/:id
  项目工作台，聚合数据源、MCP、任务、产物

/projects/:id/mcp
  MCP Server 和 Tool 管理

/projects/:id/invocations
  Tool 调用记录

/projects/:id/jobs
  任务状态、Worker 心跳、阶段进度

/projects/:id/artifacts
  产物预览、文件列表、下载

/settings/team
  成员与角色

/settings/tokens
  API Token
```

## 开发阶段

### 阶段 1：基础权限与 Workspace

目标：让系统具备最小产品边界。

任务：

- 增加 users、workspaces、workspace_members、api_tokens 表。
- 增加简单 token 鉴权 middleware。
- 给项目、任务、产物增加 workspace_id。
- 前端增加当前用户和 workspace 展示。
- 写入基础 audit log。

验收：

```text
带 token 的用户可以访问自己的 workspace。
viewer 只能看，member 可以运行任务，owner 可以管理 token。
```

### 阶段 2：MCP Registry

目标：把 MCP Server、Tools、Resources、Prompts 做成可管理资产。

任务：

- 增加 mcp_servers、mcp_tools、mcp_resources、mcp_prompts 表。
- 增加 MCP Server 创建、列表、详情接口。
- 增加 Tool 列表和 schema 展示。
- 支持手动创建 Tool 记录。
- 预留 sync 接口，后续连接真实 MCP Server 做发现。

验收：

```text
用户能在项目中注册 MCP Server，并看到可用 Tool 和 schema。
```

### 阶段 3：Tool Invocation

目标：页面可以测试调用 Tool，并记录调用全过程。

任务：

- 增加 tool_invocations 表。
- 增加 invoke 接口。
- 短任务同步返回。
- 长任务创建 Job。
- 调用结果、错误、耗时入库。
- 前端增加 Tool 调用测试面板。

验收：

```text
点击调用 Tool 后，页面能看到本次调用的输入、状态、输出或错误。
```

### 阶段 4：Job 状态与 Worker 可见性

目标：让运行状态和 Worker 状态完整返回页面。

任务：

- jobs 增加 stage、progress、message。
- 增加 job_events 表。
- Worker 上报阶段进度。
- 前端 active polling 或 SSE 展示 job 状态。
- 任务区展示 worker_id、heartbeat、attempts、error。

验收：

```text
任务从 pending 到 running 到 completed 的全过程在页面可见。
Worker 卡住或失败时，页面能看到最后阶段和错误。
```

### 阶段 5：Artifact 产物与下载

目标：所有 MCP 调用结果都有可用产物。

任务：

- 增加 artifacts、artifact_files 表。
- Worker 生成产物文件后登记 artifact_files。
- Go API 提供文件下载接口。
- 前端产物区展示文件列表、大小、hash、下载按钮。
- 支持 RAG ZIP、JSONL、manifest、quality_report。

验收：

```text
工具调用完成后，用户能在页面下载可直接使用的产物。
```

### 阶段 6：审计与操作日志

目标：产品能追踪谁调用了什么、访问了什么、下载了什么。

任务：

- 增加 audit_logs 表。
- 写操作、工具调用、产物下载写入审计。
- 前端显示项目级审计日志。
- 支持按 action、user、target_type 过滤。

验收：

```text
可以回答：谁在什么时候调用了哪个 Tool，输入是什么，生成了什么产物，是否下载。
```

### 阶段 7：产品整理与稳定性

目标：达到可持续迭代状态。

任务：

- 补齐 README 使用流程。
- 补齐 API 示例。
- 增加端到端 smoke script。
- 增加关键接口测试。
- 统一错误提示和空状态。
- 检查单文件行数不超过 500-700 行。

验收：

```text
新环境可以按文档启动、注册 MCP、调用 Tool、查看任务、下载产物。
```

## 执行约束

- 严格按阶段顺序开发，不跳过 Workspace、Invocation、Job、Artifact 这些基础闭环。
- 商业化能力暂不进入开发范围。
- 权限只做 owner/member/viewer，不引入复杂策略系统。
- 每个阶段完成后必须能跑通对应验收。
- 每个阶段完成后更新文档和 README。
- 每次提交信息使用中文。
- 单文件代码尽量控制在 500 行以内，最多不超过 700 行。
- 涉及密钥、token、外部 MCP 地址时只保存脱敏信息。

## 与数据集工作台的关系

`slm-dataset-engine` 的数据集流程可以作为 MCP 产品化后的一个业务产品线：

```text
MCP 产品底座
  -> 数据集处理 Tool
  -> RAG 导出 Tool
  -> 样本审核 Tool
  -> 质量报告 Tool
```

因此后续开发应优先抽象出：

- Invocation。
- Job 状态。
- Artifact 下载。
- Audit Log。

这些能力既服务 MCP 产品，也服务当前数据集工作台。

