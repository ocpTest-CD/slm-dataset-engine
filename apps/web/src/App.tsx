import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  api,
  DatasetVersion,
  DatasetVersionFile,
  Job,
  Project,
  QualityIssue,
  Run,
  Sample,
  SampleVersion,
  Source
} from "./api/client";

type LoadState = "idle" | "loading" | "error";
type SampleEdit = {
  sampleId: string;
  input_text: string;
  output_text: string;
  change_reason: string;
  status: string;
};

const ACTIVE_JOB_STATUS = new Set(["pending", "claimed", "running"]);
const ACTIVE_RUN_STATUS = new Set(["queued", "running"]);
const ACTIVE_VERSION_STATUS = new Set(["building"]);

export function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [samples, setSamples] = useState<Sample[]>([]);
  const [issues, setIssues] = useState<QualityIssue[]>([]);
  const [versions, setVersions] = useState<DatasetVersion[]>([]);
  const [versionFiles, setVersionFiles] = useState<Record<string, DatasetVersionFile[]>>({});
  const [sampleVersions, setSampleVersions] = useState<SampleVersion[]>([]);
  const [sampleStatus, setSampleStatus] = useState("");
  const [editingSample, setEditingSample] = useState<SampleEdit | null>(null);
  const [state, setState] = useState<LoadState>("idle");
  const [message, setMessage] = useState("");

  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId),
    [projects, selectedProjectId]
  );
  const latestRun = runs[0];
  const issueMap = useMemo(() => groupIssuesBySample(issues), [issues]);
  const activeJobs = useMemo(() => jobs.filter((job) => ACTIVE_JOB_STATUS.has(job.status)), [jobs]);
  const shouldPoll = useMemo(
    () =>
      activeJobs.length > 0 ||
      runs.some((run) => ACTIVE_RUN_STATUS.has(run.status)) ||
      versions.some((version) => ACTIVE_VERSION_STATUS.has(version.status)),
    [activeJobs.length, runs, versions]
  );

  useEffect(() => {
    void refreshProjects();
  }, []);

  useEffect(() => {
    if (selectedProjectId) {
      void refreshProjectData(selectedProjectId);
    }
  }, [selectedProjectId, sampleStatus]);

  useEffect(() => {
    if (!selectedProjectId || !shouldPoll) return;
    const timer = window.setInterval(() => {
      void refreshProjectData(selectedProjectId, false);
    }, 2500);
    return () => window.clearInterval(timer);
  }, [selectedProjectId, shouldPoll, sampleStatus]);

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

  async function refreshProjectData(projectId: string, showMessage = false) {
    const [nextSources, nextRuns, nextJobs, nextSamples, nextIssues, nextVersions] = await Promise.all([
      api.listSources(projectId),
      api.listRuns(projectId),
      api.listJobs(projectId),
      api.listSamples(projectId, sampleStatus),
      api.listQualityIssues(projectId),
      api.listVersions(projectId)
    ]);
    const fileEntries = await Promise.all(
      nextVersions.map(async (version) => [version.id, await api.listVersionFiles(version.id)] as const)
    );
    setSources(nextSources);
    setRuns(nextRuns);
    setJobs(nextJobs);
    setSamples(nextSamples);
    setIssues(nextIssues);
    setVersions(nextVersions);
    setVersionFiles(Object.fromEntries(fileEntries));
    if (showMessage) setMessage("项目数据已刷新");
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

  async function openSample(sample: Sample) {
    setEditingSample({
      sampleId: sample.id,
      input_text: sample.input_text,
      output_text: sample.output_text,
      change_reason: "",
      status: sample.status === "rejected" ? "pending_review" : sample.status
    });
    setSampleVersions(await api.listSampleVersions(sample.id));
  }

  async function saveSample(nextStatus?: string) {
    if (!selectedProjectId || !editingSample) return;
    const status = nextStatus ?? editingSample.status;
    const sample = await runAction(
      () => api.editSample(editingSample.sampleId, { ...editingSample, status }),
      status === "accepted" ? "样本已编辑并接受" : "样本已保存"
    );
    setEditingSample({
      sampleId: sample.id,
      input_text: sample.input_text,
      output_text: sample.output_text,
      change_reason: "",
      status: sample.status
    });
    setSampleVersions(await api.listSampleVersions(sample.id));
    await refreshProjectData(selectedProjectId);
  }

  async function createVersion() {
    if (!selectedProjectId) return;
    const name = `rag-${new Date().toISOString().slice(0, 10)}`;
    await runAction(() => api.createVersion(selectedProjectId, latestRun?.id ?? "", name), "RAG 导出任务已创建");
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
            <input name="name" placeholder="如 rag-knowledge-base" required />
          </label>
          <label>
            领域
            <input name="domain" placeholder="编程 / 阅读 / RAG" />
          </label>
          <label>
            目标
            <textarea name="description" rows={3} placeholder="这批数据要改善什么能力" />
          </label>
          <button type="submit" disabled={state === "loading"}>新建项目</button>
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
          <button onClick={() => selectedProjectId && refreshProjectData(selectedProjectId, true)}>刷新</button>
        </header>

        {message && <div className={state === "error" ? "notice error" : "notice"}>{message}</div>}

        <section className="metrics">
          <Metric label="数据源" value={sources.length} />
          <Metric label="运行" value={runs.length} />
          <Metric label="任务" value={jobs.length} />
          <Metric label="样本" value={samples.length} />
          <Metric label="版本" value={versions.length} />
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
              <h2>运行状态</h2>
              <button onClick={createVersion} disabled={!latestRun}>导出 RAG ZIP</button>
            </div>
            <div className="list">
              {jobs.map((job) => (
                <article key={job.id} className="job-row">
                  <div className="job-head">
                    <strong>{job.job_type}</strong>
                    <span className={`status ${job.status}`}>{job.status}</span>
                  </div>
                  <div className="progress-track"><span style={{ width: `${job.progress || 0}%` }} /></div>
                  <p>{job.stage || "queued"} · {job.message || job.error_message || "等待处理"} · {job.progress || 0}%</p>
                </article>
              ))}
              {runs.map((run) => (
                <article key={run.id} className="row compact">
                  <div>
                    <strong>Run {run.status}</strong>
                    <span>{run.total_samples} 样本 · {run.accepted_samples} 接受 · {run.issue_count} 问题</span>
                  </div>
                </article>
              ))}
              {jobs.length === 0 && runs.length === 0 && <p className="empty">点击数据源的运行按钮后，这里会显示 Worker 状态。</p>}
            </div>
          </div>
        </section>

        <section className="sample-panel">
          <div className="panel-title">
            <h2>样本审核</h2>
            <select value={sampleStatus} onChange={(event) => setSampleStatus(event.target.value)}>
              <option value="">全部状态</option>
              <option value="pending_review">待审核</option>
              <option value="edited">已编辑</option>
              <option value="accepted">已接受</option>
              <option value="rejected">已拒绝</option>
            </select>
          </div>
          <div className="review-layout">
            <div className="sample-table">
              {samples.map((sample) => {
                const sampleIssues = issueMap[sample.id] ?? [];
                return (
                  <article key={sample.id} className="sample-row">
                    <div>
                      <span className={`status ${sample.status}`}>{sample.status}</span>
                      {sampleIssues.length > 0 && <span className="issue-pill">{sampleIssues.length} 个问题</span>}
                      <strong>{truncate(sample.input_text, 150)}</strong>
                      <p>{truncate(sample.output_text || "无输出字段", 180)}</p>
                    </div>
                    <div className="sample-actions">
                      <span>{sample.quality_score ?? "-"} 分</span>
                      <button onClick={() => openSample(sample)}>编辑</button>
                      <button onClick={() => reviewSample(sample.id, "accepted")}>接受</button>
                      <button className="secondary" onClick={() => reviewSample(sample.id, "rejected")}>拒绝</button>
                    </div>
                  </article>
                );
              })}
              {samples.length === 0 && <p className="empty">运行完成后，样本会出现在这里。</p>}
            </div>

            {editingSample && (
              <aside className="editor">
                <div className="panel-title">
                  <h3>人工干预</h3>
                  <button className="secondary" onClick={() => setEditingSample(null)}>关闭</button>
                </div>
                <label>
                  输入
                  <textarea
                    rows={7}
                    value={editingSample.input_text}
                    onChange={(event) => setEditingSample({ ...editingSample, input_text: event.target.value })}
                  />
                </label>
                <label>
                  输出
                  <textarea
                    rows={9}
                    value={editingSample.output_text}
                    onChange={(event) => setEditingSample({ ...editingSample, output_text: event.target.value })}
                  />
                </label>
                <label>
                  修改说明
                  <input
                    value={editingSample.change_reason}
                    placeholder="如补齐答案、修复格式、删除噪声"
                    onChange={(event) => setEditingSample({ ...editingSample, change_reason: event.target.value })}
                  />
                </label>
                <div className="editor-actions">
                  <button onClick={() => saveSample("edited")}>保存</button>
                  <button onClick={() => saveSample("accepted")}>保存并接受</button>
                  <button className="secondary" onClick={() => saveSample("rejected")}>保存并拒绝</button>
                </div>
                <div className="issue-list">
                  {(issueMap[editingSample.sampleId] ?? []).map((issue) => (
                    <p key={issue.id}>{issue.severity} · {issue.message}</p>
                  ))}
                  {sampleVersions.map((version) => (
                    <p key={version.id}>v{version.version} · {version.edited_by} · {version.change_reason || "无说明"}</p>
                  ))}
                </div>
              </aside>
            )}
          </div>
        </section>

        <section className="sample-panel">
          <div className="panel-title">
            <h2>版本与下载</h2>
            <button onClick={createVersion} disabled={!latestRun}>创建 RAG 版本</button>
          </div>
          <div className="list">
            {versions.map((version) => (
              <article key={version.id} className="version-card">
                <div>
                  <strong>{version.version_name}</strong>
                  <span>{version.status} · {version.sample_count} 样本</span>
                </div>
                <div className="download-grid">
                  {(versionFiles[version.id] ?? []).map((file) => (
                    <a key={file.id} href={api.versionFileDownloadUrl(file.id)}>{file.file_name} · {formatBytes(file.byte_size)}</a>
                  ))}
                  {(versionFiles[version.id] ?? []).length === 0 && <span>导出完成后生成 RAG ZIP 和 JSONL 文件。</span>}
                </div>
              </article>
            ))}
            {versions.length === 0 && <p className="empty">接受样本后创建版本，系统会生成可导入 RAG 的 ZIP 包。</p>}
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

function groupIssuesBySample(issues: QualityIssue[]) {
  return issues.reduce<Record<string, QualityIssue[]>>((groups, issue) => {
    if (!issue.sample_id) return groups;
    groups[issue.sample_id] = [...(groups[issue.sample_id] ?? []), issue];
    return groups;
  }, {});
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
