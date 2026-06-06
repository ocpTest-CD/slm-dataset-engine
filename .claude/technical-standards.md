# 技术栈规范

## 总体原则

- Go 是控制面，Python 是执行面，React + Vite 是工作台界面。
- 不在 Go 中嵌入 Python 解释器；通过任务协议、数据库、队列或 HTTP/gRPC 通信。
- 业务状态以 Go 后端和数据库为准，Python Worker 不直接承担产品状态机。
- 数据处理代码必须可复现，所有流程输入、参数、版本和输出摘要都要可记录。
- 单文件代码尽量不超过 500 行，确有必要时最多不超过 700 行。

## 前端：React + Vite + TypeScript

- 使用 React + Vite + TypeScript 构建工作台。
- 优先组件化拆分，页面、数据请求、表格列定义、表单 schema、图表配置不要堆在同一个文件。
- 工作台界面应信息密度适中，避免营销式落地页。
- 样本审核表格优先使用成熟表格库，例如 TanStack Table。
- 服务端状态优先使用 TanStack Query。
- 轻量 UI 状态可使用 Zustand 或 React context。
- 流程图能力可后置，确需使用时优先考虑 React Flow。
- 图表可使用 ECharts 或 Recharts，报告图表必须能解释数据质量问题。
- 所有 API 类型尽量从 OpenAPI、schema 或共享类型生成，减少手写漂移。

前端目录建议：

```text
apps/web/src/
├── app/
├── pages/
├── features/
├── components/
├── api/
├── hooks/
├── lib/
└── styles/
```

## 后端：Go 控制面

- Go 后端负责 API、状态机、任务编排、权限、审计、版本和 manifest。
- HTTP 框架可选择 Chi、Gin 或 Echo；优先选择简单、可维护、团队熟悉的方案。
- 数据访问建议使用 sqlc、Ent 或清晰 repository 层，避免散落手写 SQL。
- API handler 不写复杂业务逻辑，业务逻辑放 service/usecase。
- Run、Job、DatasetVersion 等状态迁移必须显式建模。
- 任务取消、重试、超时和失败原因必须可追踪。
- 对上传文件、导出包和数据源路径做权限和路径安全检查。

Go 目录建议：

```text
services/api/
├── cmd/server/
├── internal/
│   ├── api/
│   ├── config/
│   ├── domain/
│   ├── service/
│   ├── repository/
│   ├── jobs/
│   └── storage/
└── migrations/
```

## Python Worker

- Python Worker 只负责执行任务，不负责产品主状态机。
- 任务入参和出参使用 pydantic 或等价 schema 校验。
- 数据处理优先使用 polars、pyarrow、duckdb 或 pandas，按任务选择。
- 模型相关能力放在独立模块：embedding、teacher、judge、tokenizer。
- 每个处理步骤输出摘要统计，不在日志中输出大量原始样本。
- AI 生成或评审必须记录模型、prompt、temperature、top_p、seed、后处理规则。
- Worker 应支持 dry-run 或小样本运行，便于调试。

Python 目录建议：

```text
workers/python/
├── slm_dataset_worker/
│   ├── jobs/
│   ├── parsers/
│   ├── processors/
│   ├── models/
│   ├── exporters/
│   ├── quality/
│   └── schemas/
└── tests/
```

## 数据库与存储

- MVP 可以使用 SQLite 降低启动成本。
- 如果涉及多用户、多并发、多环境部署，优先 PostgreSQL。
- 原始文件、导出包、报告和中间产物不直接塞进数据库，使用本地 artifacts 或对象存储。
- 数据库保存文件路径、hash、大小、mime、来源和版本信息。
- migrations 必须可回放，不手动修改已发布迁移。

核心实体建议：

```text
Project
Source
Recipe
Run
Job
Sample
SampleVersion
ReviewRecord
QualityIssue
DatasetVersion
LineageEvent
ExportArtifact
EvalLink
```

## Job 与流程

- MVP 使用 Job Table 即可，不急于引入复杂编排。
- Job 必须有明确状态：pending、claimed、running、failed、succeeded、canceled。
- Job 必须记录 attempts、max_attempts、claimed_by、started_at、finished_at、error_message。
- 长任务应定期写入 progress 和 heartbeat。
- Worker 执行任务要尽量幂等，重复执行不能破坏已有数据集版本。
- 导出版本一旦完成，应尽量不可变。

## API 规范

- API 返回结构保持稳定，错误码可读。
- 列表接口支持分页、过滤和排序。
- 样本列表避免一次返回过多 content，必要时做预览字段。
- 文件上传、导出下载、任务取消等操作需要明确权限和状态校验。
- 对外暴露的 schema 应能生成前端类型。

## 测试规范

- Go：单元测试覆盖状态机、service、repository 关键路径。
- Python：测试 parser、processor、exporter 和质量规则。
- 前端：测试关键组件、表单校验和 API 状态分支。
- 集成测试至少覆盖导入、运行、审核、导出这一条主流程。
- 数据测试使用最小脱敏夹具，不使用真实私有数据。
