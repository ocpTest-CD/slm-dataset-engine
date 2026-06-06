import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  api,
  Artifact,
  ArtifactFile,
  AuditLog,
  DatasetVersion,
  Job,
  MCPServer,
  MCPTool,
  Me,
  Project,
  QualityIssue,
  Run,
  Sample,
  Source,
  ToolInvocation
} from "./api/client";

type LoadState = "idle" | "loading" | "error";

export function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [me, setMe] = useState<Me | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [samples, setSamples] = useState<Sample[]>([]);
  const [issues, setIssues] = useState<QualityIssue[]>([]);
  const [versions, setVersions] = useState<DatasetVersion[]>([]);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpTools, setMcpTools] = useState<MCPTool[]>([]);
  const [invocations, setInvocations] = useState<ToolInvocation[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [artifactFiles, setArtifactFiles] = useState<Record<string, ArtifactFile[]>>({});
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [sampleStatus, setSampleStatus] = useState("");
  const [mcpInput, setMcpInput] = useState('{"text":"生成一个可下载 MCP 产物","metadata":{"source":"workbench"}}');
  const [state, setState] = useState<LoadState>("idle");
  const [message, setMessage] = useState("");

  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId),
    [projects, selectedProjectId]
  );
  const latestRun = runs[0];
  const hasActiveJobs = jobs.some((job) => ["pending", "claimed", "running"].includes(job.status));

  useEffect(() => {
    void refreshMe();
    void refreshProjects();
  }, []);

  useEffect(() => {
    if (selectedProjectId) {
      void refreshProjectData(selectedProjectId);
    }
  }, [selectedProjectId, sampleStatus]);

  useEffect(() => {
    if (!selectedProjectId || !hasActiveJobs) return;
    const timer = window.setInterval(() => {
      void refreshProjectData(selectedProjectId, false);
    }, 1800);
    return () => window.clearInterval(timer);
  }, [selectedProjectId, hasActiveJobs, sampleStatus]);

  async function runAction<T>(action: () => Promise<T>, nextMessage: string) {
    setState("loading");
    setMessage("");
    try {
      const result = await action();
      setMessage(nextMessage);
      return result;
    } catch (error) {
      setState("error");
      setMessage(error instanceof Error ? error.message : "操作失败");
      throw error;
    } finally {
      setState("idle");
    }
  }

  async function refreshProjects() {
    const result = await runAction(() => api.listProjects(), "项目列表已刷新");
    setProjects(result);
    if (!selectedProjectId && result.length > 0) {
      setSelectedProjectId(result[0].id);
    }
  }

  async function refreshMe() {
    const result = await api.me();
    setMe(result);
  }

  async function refreshProjectData(projectId: string, showMessage = false) {
    const [nextSources, nextRuns, nextSamples, nextIssues, nextVersions, nextServers, nextInvocations, nextJobs, nextArtifacts, nextAuditLogs] = await Promise.all([
      api.listSources(projectId),
      api.listRuns(projectId),
      api.listSamples(projectId, sampleStatus),
      api.listQualityIssues(projectId),
      api.listVersions(projectId),
      api.listMCPServers(projectId),
      api.listInvocations(projectId),
      api.listJobs(projectId),
      api.listArtifacts(projectId),
      api.listAuditLogs(projectId)
    ]);
    const nextTools = (await Promise.all(nextServers.map((server) => api.listMCPTools(server.id)))).flat();
    const nextFilesEntries = await Promise.all(
      nextArtifacts.map(async (artifact) => [artifact.id, await api.listArtifactFiles(artifact.id)] as const)
    );
    setSources(nextSources);
    setRuns(nextRuns);
    setSamples(nextSamples);
    setIssues(nextIssues);
    setVersions(nextVersions);
    setMcpServers(nextServers);
    setMcpTools(nextTools);
    setInvocations(nextInvocations);
    setJobs(nextJobs);
    setArtifacts(nextArtifacts);
    setArtifactFiles(Object.fromEntries(nextFilesEntries));
    setAuditLogs(nextAuditLogs);
    if (showMessage) setMessage("工作台状态已刷新");
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const project = await runAction(
      () =>
        api.createProject({
          name: String(form.get("name") ?? ""),
          description: String(form.get("description") ?? ""),
          domain: String(form.get("domain") ?? "")
        }),
      "项目已创建"
    );
    event.currentTarget.reset();
    setProjects((current) => [project, ...current]);
    setSelectedProjectId(project.id);
  }

  async function uploadSource(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProjectId) return;
    const form = new FormData(event.currentTarget);
    const file = form.get("file");
    if (!(file instanceof File)) return;
    await runAction(() => api.uploadSource(selectedProjectId, file), "数据源已上传");
    event.currentTarget.reset();
    await refreshProjectData(selectedProjectId);
  }

  async function createMCPServer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProjectId) return;
    const form = new FormData(event.currentTarget);
    await runAction(
      () =>
        api.createMCPServer(selectedProjectId, {
          name: String(form.get("name") ?? ""),
          endpoint: String(form.get("endpoint") ?? ""),
          transport: String(form.get("transport") ?? "stdio")
        }),
      "MCP Server 已注册，并自动创建 echo_artifact Tool"
    );
    event.currentTarget.reset();
    await refreshProjectData(selectedProjectId);
  }

  async function invokeTool(toolId: string) {
    if (!selectedProjectId) return;
    await runAction(() => api.invokeMCPTool(toolId, mcpInput), "Tool 调用已进入任务队列");
    await refreshProjectData(selectedProjectId);
  }

  async function startRun(sourceId: string) {
    if (!selectedProjectId) return;
    await runAction(() => api.startRun(sourceId), "Run 已创建，等待 Worker 处理");
    await refreshProjectData(selectedProjectId);
  }

  async function reviewSample(sampleId: string, status: string) {
    if (!selectedProjectId) return;
    await runAction(() => api.reviewSample(sampleId, status), "样本审核状态已更新");
    await refreshProjectData(selectedProjectId);
  }

  async function createVersion() {
    if (!selectedProjectId) return;
    const name = `dataset-${new Date().toISOString().slice(0, 10)}`;
    await runAction(() => api.createVersion(selectedProjectId, latestRun?.id ?? "", name), "导出任务已创建");
    await refreshProjectData(selectedProjectId);
  }

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">D</span>
          <div>
            <strong>SLM Dataset Engine</strong>
            <small>数据集流程工作台</small>
          </div>
        </div>

        <form className="create-form" onSubmit={createProject}>
          <label>
            项目名称
            <input name="name" placeholder="如 react-hooks-sft" required />
          </label>
          <label>
            领域
            <input name="domain" placeholder="编程 / 阅读 / RAG" />
          </label>
          <label>
            目标
            <textarea name="description" rows={3} placeholder="这批数据要改善什么能力" />
          </label>
          <button type="submit" disabled={state === "loading"}>
            新建项目
          </button>
        </form>

        <nav className="project-list" aria-label="项目列表">
          {projects.map((project) => (
            <button
              key={project.id}
              className={project.id === selectedProjectId ? "project active" : "project"}
              onClick={() => setSelectedProjectId(project.id)}
            >
              <strong>{project.name}</strong>
              <span>{project.domain || "未设置领域"}</span>
            </button>
          ))}
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Workbench</p>
            <h1>{selectedProject?.name ?? "创建一个数据集项目"}</h1>
          </div>
          <div className="topbar-actions">
            {me && <span>{me.workspace.name} · {me.principal.role}</span>}
            <button onClick={() => selectedProjectId && refreshProjectData(selectedProjectId, true)}>刷新</button>
          </div>
        </header>

        {message && <div className={state === "error" ? "notice error" : "notice"}>{message}</div>}

        <section className="metrics">
          <Metric label="数据源" value={sources.length} />
          <Metric label="运行" value={runs.length} />
          <Metric label="样本" value={samples.length} />
          <Metric label="质量问题" value={issues.length} />
          <Metric label="版本" value={versions.length} />
          <Metric label="MCP Tool" value={mcpTools.length} />
          <Metric label="产物" value={artifacts.length} />
        </section>

        <section className="panels">
          <div className="panel">
            <div className="panel-title">
              <h2>MCP Registry</h2>
              <form className="upload-form" onSubmit={createMCPServer}>
                <input name="name" placeholder="Server 名称" />
                <input name="endpoint" placeholder="endpoint / stdio 命令" />
                <input name="transport" placeholder="stdio" />
                <button type="submit" disabled={!selectedProjectId}>注册</button>
              </form>
            </div>
            <div className="list">
              {mcpServers.map((server) => (
                <article key={server.id} className="row">
                  <div>
                    <strong>{server.name}</strong>
                    <span>{server.transport} · {server.status} · {server.endpoint || "local"}</span>
                  </div>
                </article>
              ))}
              {mcpServers.length === 0 && <p className="empty">注册 MCP Server 后，系统会创建一个可验证闭环的 echo_artifact Tool。</p>}
            </div>
          </div>

          <div className="panel">
            <div className="panel-title">
              <h2>Tool 调用</h2>
              <button onClick={() => selectedProjectId && refreshProjectData(selectedProjectId)}>同步状态</button>
            </div>
            <textarea className="json-input" value={mcpInput} onChange={(event) => setMcpInput(event.target.value)} rows={4} />
            <div className="list">
              {mcpTools.map((tool) => (
                <article key={tool.id} className="row">
                  <div>
                    <strong>{tool.name}</strong>
                    <span>{tool.description || "无描述"}</span>
                  </div>
                  <button onClick={() => invokeTool(tool.id)} disabled={!tool.enabled}>调用</button>
                </article>
              ))}
              {invocations.slice(0, 4).map((invocation) => (
                <article key={invocation.id} className="row muted">
                  <div>
                    <strong>{invocation.tool_name || invocation.tool_id}</strong>
                    <span>{invocation.status} · job {shortId(invocation.job_id)}</span>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="panels">
          <div className="panel">
            <div className="panel-title">
              <h2>任务状态</h2>
              {hasActiveJobs && <span className="live-dot">自动刷新中</span>}
            </div>
            <div className="list">
              {jobs.map((job) => (
                <article key={job.id} className="row">
                  <div>
                    <strong>{job.job_type} · {job.status}</strong>
                    <span>{job.stage || "queued"} · {job.progress}% · {job.message || "无消息"}</span>
                    {job.error_message && <span className="danger-text">{job.error_message}</span>}
                  </div>
                </article>
              ))}
              {jobs.length === 0 && <p className="empty">Tool 调用、数据导入和导出任务会出现在这里。</p>}
            </div>
          </div>

          <div className="panel">
            <div className="panel-title">
              <h2>产物下载</h2>
            </div>
            <div className="list">
              {artifacts.map((artifact) => (
                <article key={artifact.id} className="row artifact-row">
                  <div>
                    <strong>{artifact.name}</strong>
                    <span>{artifact.artifact_type} · {artifact.status}</span>
                    {(artifactFiles[artifact.id] ?? []).map((file) => (
                      <a key={file.id} href={api.artifactFileDownloadUrl(file.id)}>
                        {file.file_name} · {formatBytes(file.byte_size)}
                      </a>
                    ))}
                  </div>
                </article>
              ))}
              {artifacts.length === 0 && <p className="empty">任务完成后，可下载产物会出现在这里。</p>}
            </div>
          </div>
        </section>

        <section className="sample-panel compact">
          <div className="panel-title">
            <h2>审计日志</h2>
          </div>
          <div className="list audit-list">
            {auditLogs.slice(0, 8).map((log) => (
              <article key={log.id} className="row">
                <div>
                  <strong>{log.action}</strong>
                  <span>{log.target_type} · {shortId(log.target_id)} · {new Date(log.created_at).toLocaleString()}</span>
                </div>
              </article>
            ))}
            {auditLogs.length === 0 && <p className="empty">关键操作、Tool 调用和下载会记录在这里。</p>}
          </div>
        </section>

        <section className="panels">
          <div className="panel">
            <div className="panel-title">
              <h2>数据源</h2>
              <form className="upload-form" onSubmit={uploadSource}>
                <input name="file" type="file" accept=".jsonl,.csv,.md,.markdown,.txt" />
                <button type="submit" disabled={!selectedProjectId}>上传</button>
              </form>
            </div>
            <div className="list">
              {sources.map((source) => (
                <article key={source.id} className="row">
                  <div>
                    <strong>{source.filename}</strong>
                    <span>{source.source_type} · {formatBytes(source.byte_size)} · {source.status}</span>
                  </div>
                  <button onClick={() => startRun(source.id)}>运行</button>
                </article>
              ))}
              {sources.length === 0 && <p className="empty">上传 JSONL、CSV 或 Markdown 后开始处理。</p>}
            </div>
          </div>

          <div className="panel">
            <div className="panel-title">
              <h2>运行与导出</h2>
              <button onClick={createVersion} disabled={!latestRun}>创建版本</button>
            </div>
            <div className="list">
              {runs.map((run) => (
                <article key={run.id} className="row">
                  <div>
                    <strong>{run.status}</strong>
                    <span>{run.total_samples} 样本 · {run.issue_count} 问题 · {run.progress}%</span>
                  </div>
                </article>
              ))}
              {versions.map((version) => (
                <article key={version.id} className="row version">
                  <div>
                    <strong>{version.version_name}</strong>
                    <span>{version.status} · {version.sample_count} 样本</span>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="sample-panel">
          <div className="panel-title">
            <h2>样本审核</h2>
            <select value={sampleStatus} onChange={(event) => setSampleStatus(event.target.value)}>
              <option value="">全部状态</option>
              <option value="pending_review">待审核</option>
              <option value="accepted">已接受</option>
              <option value="rejected">已拒绝</option>
            </select>
          </div>
          <div className="sample-table">
            {samples.map((sample) => (
              <article key={sample.id} className="sample-row">
                <div>
                  <span className={`status ${sample.status}`}>{sample.status}</span>
                  <strong>{truncate(sample.input_text, 150)}</strong>
                  <p>{truncate(sample.output_text || "无输出字段", 180)}</p>
                </div>
                <div className="sample-actions">
                  <span>{sample.quality_score ?? "-"} 分</span>
                  <button onClick={() => reviewSample(sample.id, "accepted")}>接受</button>
                  <button onClick={() => reviewSample(sample.id, "rejected")}>拒绝</button>
                </div>
              </article>
            ))}
            {samples.length === 0 && <p className="empty">运行完成后，样本会出现在这里。</p>}
          </div>
        </section>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function truncate(value: string, max: number) {
  if (value.length <= max) return value;
  return `${value.slice(0, max)}...`;
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function shortId(value: string) {
  return value ? value.slice(0, 8) : "-";
}
