# slm-dataset-engine

`slm-dataset-engine` 是面向小模型研发的数据集流程工作台。它的目标不是堆叠清洗脚本，而是把数据集接入、处理、审核、质检、版本、导出和评测闭环变成可复现、可追溯、可协作的工程系统。

当前项目处于初始化阶段，详细技术方案、架构图和协作规范保存在 `.claude/` 目录中。

## 项目目标

这个项目重点解决以下问题：

- 数据从哪里来，是否有授权，是否可用于训练或评测。
- 数据经历了哪些解析、清洗、切分、过滤、改写、审核和导出步骤。
- 每条样本为什么被接受、拒绝、修改或标记为风险样本。
- AI 辅助生成或评审时使用了哪个 teacher/judge 模型、prompt 和采样参数。
- 每个数据集版本的 manifest、质量报告、血缘和导出物如何保存。
- 数据集版本如何与模型训练、蒸馏、微调和评测结果关联。

核心原则：

> 数据集研发必须可复现、可追溯、可评测、可导出。

## 技术路线

推荐采用 Go 控制面 + Python 执行面：

- 前端：React + Vite + TypeScript，负责工作台、样本审核、流程配置、运行状态和质量报告。
- 后端：Go，负责 API、项目管理、任务编排、状态机、权限、审计、版本和 manifest。
- Worker：Python，负责文档解析、数据处理、模型调用、embedding、质量评估和导出。
- 存储：MVP 可使用 SQLite；多用户、多并发或部署环境复杂后迁移 PostgreSQL。
- 产物：原始文件、导出包、报告和中间产物放入 `artifacts/` 或对象存储，默认不提交 Git。

架构图和完整方案见 [.claude/architecture.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.claude/architecture.md)。

## MVP 范围

第一版优先打通数据闭环，不做复杂流程画布：

1. 创建数据集项目。
2. 导入 JSONL、CSV、Markdown。
3. 配置基础 recipe：解析、清洗、去重、长度过滤、字段校验。
4. 运行流程并记录 Run。
5. 生成 Sample、QualityIssue 和统计摘要。
6. 在前端表格中审核样本。
7. 导出 DatasetVersion、JSONL、manifest 和 quality report。

第二阶段再加入 AI 辅助生成、embedding、judge 评分、评测关联和更复杂的流程模板。

## 推荐目录结构

后续按需创建：

```text
slm-dataset-engine/
├── AGENTS.md
├── README.md
├── .claude/
├── apps/
│   └── web/               # React + Vite 工作台
├── services/
│   └── api/               # Go 后端控制面
├── workers/
│   └── python/            # Python 数据处理与模型 Worker
├── packages/
│   └── shared/            # 共享 schema、协议或生成代码
├── configs/               # recipe、环境样例和运行配置
├── migrations/            # 数据库迁移
├── scripts/               # 本地开发、导入导出和维护脚本
├── tests/                 # 跨模块测试和样例夹具
├── docs/                  # 项目内工程文档
└── artifacts/             # 本地生成物入口，默认不提交
```

## 核心实体

建议围绕以下实体建设：

- `Project`：数据集项目。
- `Source`：数据来源、授权、hash、导入记录。
- `Recipe`：流程配置和参数。
- `Run`：一次流程运行。
- `Job`：Worker 可执行任务。
- `Sample`：样本主体。
- `SampleVersion`：样本修改历史。
- `ReviewRecord`：人工审核记录。
- `QualityIssue`：质量问题和风险标记。
- `DatasetVersion`：导出的数据集版本。
- `LineageEvent`：样本血缘事件。
- `ExportArtifact`：导出文件、报告和 manifest。
- `EvalLink`：数据集版本与评测结果的关联。

## 协作入口

AI 或协作者进入项目后，应先读取：

1. [AGENTS.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/AGENTS.md)
2. [.claude/README.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.claude/README.md)

再按任务类型按需读取：

- 架构和模块边界：[.claude/architecture.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.claude/architecture.md)
- 技术栈规范：[.claude/technical-standards.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.claude/technical-standards.md)
- 协作规范：[.claude/collaboration.md](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.claude/collaboration.md)

## 研发约束

- 提交信息使用中文。
- 每个研发文件尽量控制在 500 行以内，确有必要时最多不超过 700 行代码。
- 不提交原始数据、私有数据集、模型权重、checkpoint、embedding 索引、大规模日志、密钥或未脱敏样本。
- 处理数据时优先输出摘要统计，避免在日志、终端或文档中暴露大量原始样本。
- AI 生成或 AI 评审的数据必须记录模型、prompt、采样参数、后处理规则和安全策略。

## 部署入口

当前仓库已提供最小生产部署骨架：

- [deploy/compose.yaml](/Users/Zhuanz/work-space/slm/slm-dataset-engine/deploy/compose.yaml)：生产服务器 Docker Compose 入口。
- [deploy/Caddyfile](/Users/Zhuanz/work-space/slm/slm-dataset-engine/deploy/Caddyfile)：Caddy healthcheck 网关配置。
- [deploy/site/index.html](/Users/Zhuanz/work-space/slm/slm-dataset-engine/deploy/site/index.html)：当前生产占位页面。
- [deploy/nginx-data.yi-flow.com.conf](/Users/Zhuanz/work-space/slm/slm-dataset-engine/deploy/nginx-data.yi-flow.com.conf)：服务器 Nginx 反向代理配置。
- [.github/workflows/deploy.yml](/Users/Zhuanz/work-space/slm/slm-dataset-engine/.github/workflows/deploy.yml)：GitHub Actions 部署流程。

初始部署提供 `/healthz` 和静态占位页面。当前服务器已有 Nginx 占用公网 80/443，因此 Compose 中的 Caddy 只监听 `127.0.0.1:18080`，公网域名由 Nginx 反向代理进入。当前 `data.yi-flow.com` DNS 尚未指向生产服务器，DNS A 记录改到服务器后再启用 HTTPS 证书配置。

## 本地开发

前端：

```bash
cd apps/web
npm install
npm run dev
```

Go API：

```bash
cd services/api
go mod tidy
go run ./cmd/server
```

Python Worker：

```bash
cd workers/python
python3.11 -m venv .venv
. .venv/bin/activate
pip install .
slm-dataset-worker
```

本地需要 PostgreSQL，并设置 `DATABASE_URL`。Worker 要求 Python 3.11 或更高版本；生产镜像使用 Python 3.12。

## 当前状态

- 已初始化项目级 `AGENTS.md`。
- 已初始化 `.claude/` 上下文目录。
- 已确定 React + Vite、Go、Python Worker 的技术路线。
- 已实现 Go API、Python Worker、React 工作台和 PostgreSQL schema 的 MVP 闭环。
- 已接入生产 Docker Compose 与 GitHub Actions 镜像部署流程。
