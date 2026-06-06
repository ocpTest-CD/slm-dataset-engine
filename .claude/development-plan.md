# 开发计划：全链路数据集工作台 MVP

- 日期：2026-06-06
- 状态：5 阶段闭环执行完成
- 目标：打通「导入数据 -> 运行流程 -> 生成样本与质量问题 -> 人工审核 -> 导出数据集版本」闭环。

## 目标边界

第一版不做复杂流程画布、不做多租户、不做插件市场、不做大模型自动生成。优先建设稳定的数据集生命周期：

- 项目管理。
- 数据源上传。
- Recipe 快照。
- Run/Job 状态机。
- Python Worker 处理 JSONL、CSV、Markdown。
- 样本与质量问题入库。
- 前端样本审核。
- DatasetVersion 导出 RAG ZIP、JSONL、manifest 和质量报告。

## 项目划分

```text
apps/web/                 React + Vite 工作台
services/api/             Go 后端控制面
workers/python/           Python 数据处理与模型 Worker
packages/contracts/       共享状态枚举、任务协议、API 草案
migrations/               PostgreSQL schema
configs/recipes/          内置流程 recipe，后续补充
tests/                    跨模块端到端夹具，后续补充
deploy/                   生产部署配置
```

## 模块职责

### Go API 控制面

- `project`：项目创建、列表、详情。
- `source`：文件上传、hash、路径、导入状态。
- `recipe`：流程配置快照。
- `run`：运行状态机、进度聚合、取消和失败记录。
- `job`：Job Table、claim、heartbeat、重试。
- `sample`：样本列表、预览、审核状态。
- `quality`：质量问题和统计报告。
- `dataset_version`：导出版本、manifest、产物路径。
- `storage`：本地 artifacts，后续兼容 R2。

### Python Worker 执行面

- `jobs`：claim job、heartbeat、写回状态。
- `parsers`：JSONL、CSV、Markdown。
- `processors`：字段规范化、长度过滤、基础清洗。
- `quality`：字段缺失、长度、重复、格式问题。
- `exporters`：JSONL、manifest、quality report。

### React + Vite 工作台

- 项目列表与创建。
- 数据源上传。
- 运行触发与运行状态。
- 样本表格、审核操作。
- 质量报告摘要。
- 数据集版本导出。

## 数据流转

```text
用户创建 Project
-> 上传 Source 文件
-> Go API 保存 artifact、计算 hash、创建 Source
-> 用户触发 Run
-> Go API 创建 RecipeSnapshot、Run、Job(import_dataset)
-> Python Worker claim Job
-> Worker 解析 Source artifact
-> Worker 写入 Sample、QualityIssue、LineageEvent、Run 统计
-> 用户在前端审核 Sample
-> 用户创建 DatasetVersion
-> Go API 创建 Job(export_dataset)
-> Worker 导出 JSONL + manifest + quality report
-> Go API 标记 DatasetVersion ready
```

## 状态机

### Source

```text
created -> uploaded -> ready
                  |-> blocked
ready -> archived
```

### Run

```text
created -> queued -> running -> waiting_review -> exporting -> completed
                    |          |                 |-> failed
                    |          |-> failed
                    |-> canceled
```

### Job

```text
pending -> claimed -> running -> succeeded
                     |-> failed -> pending
                     |-> failed_final
pending/running -> canceled
claimed/running -> stale -> pending
```

### Sample

```text
imported -> quality_checked -> pending_review
pending_review -> accepted
pending_review -> rejected
pending_review -> edited -> pending_review
accepted -> exported
```

### DatasetVersion

```text
draft -> building -> ready -> published
                  |-> failed
published -> deprecated
```

## 阶段计划

1. 工程骨架
   - 初始化 Go API、Python Worker、React + Vite。
   - 提供本地启动和构建命令。

2. 数据库 schema
   - 创建核心实体表。
   - 建立状态、索引和基础约束。

3. Go API MVP
   - 项目、上传、运行、样本审核、版本创建接口。
   - Job Table claim 接口供 Worker 使用。

4. Python Worker MVP
   - 支持 JSONL、CSV、Markdown。
   - 支持 import/export 两类 Job。
   - 输出摘要统计。

5. 前端工作台 MVP
   - 项目列表。
   - 上传数据源。
   - 触发运行。
   - 样本审核。
   - 版本导出。

6. 部署与验证
   - Docker Compose 增加 API、Worker、Web。
   - GitHub Actions 构建镜像并部署。
   - 验证生产 healthcheck 和占位/工作台页面。

## 2026-06-07 五阶段闭环增强

### 阶段 1：Job / Runner 状态可见

- `jobs` 增加 `stage`、`progress`、`message`。
- 新增 `job_events` 记录 Worker progress 事件。
- 前端轮询项目内 Job、Run、Version，展示阶段、进度条和错误消息。

### 阶段 2：样本人工干预

- 样本审核支持编辑输入、输出和修改说明。
- 每次编辑写入 `sample_versions`。
- 编辑后可保存为 `edited`，也可直接保存并接受或拒绝。

### 阶段 3：RAG ZIP 导出

- DatasetVersion 导出只收集 `accepted` 样本。
- Worker 生成 `rag-dataset.zip`、`chunks.jsonl`、`documents.jsonl`、`dataset.jsonl`、`manifest.json`、`quality_report.json` 和包内 `README.md`。
- 新增 `dataset_version_files` 保存文件路径、大小、MIME 和 sha256。
- 前端展示版本文件并提供下载入口。

### 阶段 4：质量问题联动样本

- `QualityIssue` API 返回 `sample_id`。
- 前端按样本聚合问题数量，并在编辑区展示问题信息。

### 阶段 5：文档和烟测

- README 补充工作台闭环、RAG 导出包结构和烟测命令。
- `scripts/smoke_dataset_workbench.py` 验证创建项目、上传、运行、编辑、接受、导出和 ZIP 文件检查。

## 验收标准

- 本地可以构建 Go API、Python Worker、React Web。
- 有 PostgreSQL 时可以跑通一条 JSONL 导入闭环。
- Worker 能把样本和质量问题写入数据库。
- 前端能查看项目、上传数据、触发运行、看到 Job 进度、编辑样本、审核样本、创建并下载 RAG 导出版本。
- RAG ZIP 内至少包含 `chunks.jsonl`、`documents.jsonl`、`dataset.jsonl`、`manifest.json`、`quality_report.json` 和 `README.md`。
- 生产部署 workflow 成功。
- 单文件代码控制在 500 行以内，最多不超过 700 行。
