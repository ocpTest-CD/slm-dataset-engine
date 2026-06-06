# 技术方案与架构

- 状态：草案
- 日期：2026-06-06
- 项目：slm-dataset-engine
- 目标：数据集流程化工作台

## 产品定位

`slm-dataset-engine` 是小模型数据集研发工作台，不是通用 ETL 平台，也不是单纯的数据清洗脚本集合。

它要解决的问题：

- 数据从哪里来、是否有授权、是否能用于训练。
- 数据经历了哪些解析、清洗、切分、改写、过滤和审核步骤。
- 每条样本为什么被接受、拒绝或修改。
- AI 生成数据时使用了哪个 teacher 模型、prompt 和采样参数。
- 导出的数据集版本和上一版本有什么差异。
- 某个数据集版本是否改善了模型评测结果。

核心目标：

> 把数据集研发流程做成可复现、可追溯、可评测、可导出的工程系统。

## 技术路线

推荐采用 Go 控制面 + Python 执行面：

- React + Vite 前端负责工作台、样本审核、流程配置、运行状态和质量报告。
- Go 后端负责 API、项目管理、任务编排、状态机、权限、审计、版本和 manifest。
- Python Worker 负责文档解析、数据处理、模型调用、embedding、judge 评分和导出。
- 存储层保存元数据、样本、质量问题、血缘、任务状态、文件路径和导出记录。

## SVG 架构图

```svg
<svg width="1180" height="720" viewBox="0 0 1180 720" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <style>
      .title { font: 700 28px Arial; fill: #111827; }
      .sub { font: 15px Arial; fill: #4b5563; }
      .box { fill: #fff; stroke: #2563eb; stroke-width: 1.6; rx: 10; }
      .go { fill: #eff6ff; stroke: #2563eb; stroke-width: 1.8; rx: 10; }
      .py { fill: #f0fdf4; stroke: #16a34a; stroke-width: 1.8; rx: 10; }
      .store { fill: #fff7ed; stroke: #ea580c; stroke-width: 1.6; rx: 10; }
      .label { font: 700 17px Arial; fill: #111827; }
      .txt { font: 13px Arial; fill: #374151; }
      .tiny { font: 12px Arial; fill: #6b7280; }
      .arrow { stroke: #374151; stroke-width: 1.8; fill: none; marker-end: url(#arrow); }
      .dash { stroke: #6b7280; stroke-width: 1.5; stroke-dasharray: 6 5; fill: none; marker-end: url(#arrow); }
    </style>
    <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto">
      <path d="M0,0 L8,3 L0,6 Z" fill="#374151"/>
    </marker>
  </defs>

  <text x="590" y="40" text-anchor="middle" class="title">SLM Dataset Engine：Go 控制面 + Python 执行面</text>
  <text x="590" y="68" text-anchor="middle" class="sub">数据集流程工作台：导入、清洗、生成、审核、质检、版本、导出、评测闭环</text>

  <rect x="40" y="110" width="240" height="500" class="box"/>
  <text x="160" y="140" text-anchor="middle" class="label">React + Vite 前端</text>
  <text x="160" y="175" text-anchor="middle" class="txt">项目工作台</text>
  <text x="160" y="205" text-anchor="middle" class="txt">流程配置</text>
  <text x="160" y="235" text-anchor="middle" class="txt">样本表格 / 审核</text>
  <text x="160" y="265" text-anchor="middle" class="txt">运行进度</text>
  <text x="160" y="295" text-anchor="middle" class="txt">质量报告</text>
  <text x="160" y="325" text-anchor="middle" class="txt">数据集版本</text>

  <rect x="340" y="100" width="310" height="540" class="go"/>
  <text x="495" y="132" text-anchor="middle" class="label">Go 后端控制面</text>
  <rect x="375" y="165" width="240" height="48" class="box"/>
  <text x="495" y="195" text-anchor="middle" class="txt">API / Auth / Project</text>
  <rect x="375" y="235" width="240" height="48" class="box"/>
  <text x="495" y="265" text-anchor="middle" class="txt">Dataset / Sample / Review</text>
  <rect x="375" y="305" width="240" height="48" class="box"/>
  <text x="495" y="335" text-anchor="middle" class="txt">Workflow Recipe 管理</text>
  <rect x="375" y="375" width="240" height="48" class="box"/>
  <text x="495" y="405" text-anchor="middle" class="txt">Run 状态机 / 任务编排</text>
  <rect x="375" y="445" width="240" height="48" class="box"/>
  <text x="495" y="475" text-anchor="middle" class="txt">进度推送 SSE / WebSocket</text>
  <rect x="375" y="515" width="240" height="48" class="box"/>
  <text x="495" y="545" text-anchor="middle" class="txt">导出 / Manifest / Lineage</text>

  <rect x="720" y="100" width="320" height="540" class="py"/>
  <text x="880" y="132" text-anchor="middle" class="label">Python 执行面 Worker</text>
  <rect x="755" y="165" width="250" height="48" class="box"/>
  <text x="880" y="195" text-anchor="middle" class="txt">文档解析 / 文本抽取</text>
  <rect x="755" y="235" width="250" height="48" class="box"/>
  <text x="880" y="265" text-anchor="middle" class="txt">清洗 / 去重 / 脱敏检测</text>
  <rect x="755" y="305" width="250" height="48" class="box"/>
  <text x="880" y="335" text-anchor="middle" class="txt">切分 / token 统计</text>
  <rect x="755" y="375" width="250" height="48" class="box"/>
  <text x="880" y="405" text-anchor="middle" class="txt">Embedding / Teacher 生成</text>
  <rect x="755" y="445" width="250" height="48" class="box"/>
  <text x="880" y="475" text-anchor="middle" class="txt">质量打分 / Judge</text>
  <rect x="755" y="515" width="250" height="48" class="box"/>
  <text x="880" y="545" text-anchor="middle" class="txt">JSONL / Parquet / HF 导出</text>

  <rect x="340" y="665" width="700" height="46" class="store"/>
  <text x="690" y="694" text-anchor="middle" class="txt">存储层：PostgreSQL 或 SQLite 元数据 + 对象存储 artifacts + 任务队列/Job Table</text>

  <path d="M280 250 H340" class="arrow"/>
  <path d="M650 400 H720" class="arrow"/>
  <path d="M720 435 H650" class="dash"/>
  <path d="M495 640 V665" class="arrow"/>
  <path d="M880 640 V665" class="arrow"/>
</svg>
```

## 模块职责

### 前端工作台

- 项目列表、数据源管理、流程配置和运行记录。
- 样本表格、人工审核、批量操作和问题标注。
- 质量报告、版本差异、导出记录和评测关联。
- 运行进度展示，支持 SSE 或 WebSocket。

### Go 后端控制面

- API、鉴权、权限、审计日志和配置管理。
- Project、Source、Recipe、Run、Sample、Review、DatasetVersion 等核心实体。
- Run 状态机、任务创建、取消、重试、超时和进度汇总。
- 数据集 manifest、lineage、quality issue 和 export 管理。
- 与 Python Worker 通过 Job Table、队列或内部 API 协作。

### Python Worker 执行面

- 文件解析：JSONL、CSV、Markdown、HTML、PDF。
- 数据处理：清洗、去重、字段修复、语言检测、token 统计。
- 模型调用：embedding、teacher 生成、judge 评分、分类和改写。
- 质量报告：分布统计、问题统计、重复率、长度分布和风险样本。
- 导出：JSONL、Parquet、HuggingFace datasets、OpenAI fine-tuning 格式。

## MVP 路线

第一版只做数据闭环，不做复杂流程画布：

1. 创建数据集项目。
2. 导入 JSONL、CSV、Markdown。
3. 配置基础 recipe：解析、清洗、去重、长度过滤、字段校验。
4. 运行流程并记录 Run。
5. 生成 Sample、QualityIssue 和统计摘要。
6. 在前端表格中审核样本。
7. 导出 DatasetVersion、JSONL、manifest 和 quality report。

第二阶段再加入 AI 辅助生成、embedding、judge 评分和评测关联。

## 任务通信建议

MVP 推荐使用 Job Table：

- Go 创建任务并写入数据库。
- Python Worker claim 待执行任务。
- Worker 写回状态、日志、统计、产物路径和错误信息。
- Go 聚合进度并推送给前端。

中期可以替换为 Redis Streams、NATS 或 RabbitMQ。复杂长流程稳定后再考虑 Temporal。
