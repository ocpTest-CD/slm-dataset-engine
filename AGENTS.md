# AGENTS.md

## 项目概述

`slm-dataset-engine` 是小模型数据集流程工作台项目，目标是把数据集研发从零散脚本、人工抽检和临时记录，升级为可复现、可追溯、可评测、可导出的工程化流程。

当前仓库是独立工程仓库，位于 `/Users/Zhuanz/work-space/slm/slm-dataset-engine`。

项目核心关注：

- 数据接入、解析、清洗、去重、脱敏检测和格式规范化。
- SFT、RAG、蒸馏、偏好数据和评测数据集的构造流程。
- 样本审核、质量门禁、问题记录和人工修订。
- Dataset Version、Manifest、Lineage、Quality Report 和导出产物管理。
- 数据集版本与模型训练、评测结果之间的闭环追踪。

推荐技术方向：

- 前端：React + Vite + TypeScript。
- 后端控制面：Go。
- 模型与数据处理执行面：Python Worker。
- 元数据存储：MVP 可使用 SQLite，后续可迁移 PostgreSQL。
- 文件与导出物：本地 `artifacts/` 或对象存储，默认不提交大体积产物。

## 必读与按需阅读规则

每次执行任务前，AI 必须先读取：

1. 当前文件 `AGENTS.md`。
2. `.claude/README.md`。

然后根据任务类型按需读取 `.claude/` 中的相关文档：

- 架构、模块划分、系统边界：读取 `.claude/architecture.md`。
- 前端、Go、Python、数据库、任务队列等实现：读取 `.claude/technical-standards.md`。
- Git、提交、评审、文件规模、协作边界：读取 `.claude/collaboration.md`。

`.claude/` 是按需阅读目录，不要求每次全量读取所有文件，但必须读取足够上下文再动手。若任务涉及多个技术栈，需要同时读取对应规范。

## 工作区边界

- 当前仓库承载 `slm-dataset-engine` 的工程代码、配置、脚本和本地项目文档。
- 不提交原始数据、私有数据集、模型权重、checkpoint、embedding 索引、大规模日志、密钥或未脱敏样本。
- 大体积数据和模型产物应放在本地目录、对象存储或外部数据仓库，并在仓库中只保留 manifest、路径索引和脱敏说明。
- 生成的临时产物、缓存、日志和导出包默认不纳入 Git。

## 推荐目录结构

后续可以按需要逐步创建：

```text
slm-dataset-engine/
├── AGENTS.md
├── .claude/
├── apps/
│   └── web/               # React + Vite 工作台
├── services/
│   └── api/               # Go 后端控制面
├── workers/
│   └── python/            # Python 数据处理与模型 Worker
├── packages/
│   └── shared/            # 共享 schema、协议或生成代码
├── configs/               # 流程 recipe、环境样例和运行配置
├── migrations/            # 数据库迁移
├── scripts/               # 本地开发、导入导出和维护脚本
├── tests/                 # 跨模块测试和样例夹具
├── docs/                  # 项目内工程文档，非大体积数据
└── artifacts/             # 本地生成物入口，默认不提交
```

## 数据工程原则

- 所有数据处理流程应尽量可复现：记录输入来源、hash、recipe、代码版本、随机种子和输出摘要。
- 不直接覆盖原始数据；清洗、切分、增强和合成数据应生成新版本。
- 每条样本应保留 provenance：来源、处理步骤、过滤原因、质量分、人工审核记录和模型生成信息。
- AI 生成或 AI 评审的数据必须记录 teacher 或 judge 模型、prompt、采样参数、后处理规则和安全策略。
- 导出数据集必须包含 manifest、统计摘要、质量报告和血缘信息。

## 研发规则

- 修改前检查当前目录结构和 Git 状态，避免覆盖用户已有改动。
- 保持改动范围小而清晰，不顺手重构无关内容。
- 新增功能优先补充最小可运行命令、样例输入或 dry-run 模式。
- 处理数据时优先输出摘要统计，避免在终端或文档中暴露大量原始样本。
- 引入新依赖时说明用途、替代方案和环境影响。
- 完成改动后尽量运行相关检查；无法运行时说明原因。
- 提交信息使用中文，格式清晰表达变更目的。
- 每个研发文件尽量控制在 500 行以内，确有必要时最多不超过 700 行代码。超过前应拆分模块、组件、handler、service 或工具函数。
