export type Project = {
  id: string;
  name: string;
  description: string;
  domain: string;
  created_at: string;
};

export type Source = {
  id: string;
  project_id: string;
  filename: string;
  source_type: string;
  status: string;
  byte_size: number;
  created_at: string;
};

export type Run = {
  id: string;
  project_id: string;
  source_id: string;
  status: string;
  progress: number;
  total_samples: number;
  accepted_samples: number;
  rejected_samples: number;
  issue_count: number;
};

export type Sample = {
  id: string;
  project_id: string;
  run_id: string;
  source_id: string;
  status: string;
  input_text: string;
  output_text: string;
  quality_score: number | null;
  token_count: number;
  created_at: string;
};

export type QualityIssue = {
  id: string;
  sample_id: string;
  issue_type: string;
  severity: string;
  message: string;
};

export type DatasetVersion = {
  id: string;
  version_name: string;
  status: string;
  artifact_path: string;
  sample_count: number;
  created_at: string;
};

export type Job = {
  id: string;
  job_type: string;
  status: string;
  stage: string;
  progress: number;
  message: string;
  claimed_by: string;
  error_message: string;
};

export type SampleVersion = {
  id: string;
  version: number;
  input_text: string;
  output_text: string;
  edited_by: string;
  change_reason: string;
  created_at: string;
};

export type DatasetVersionFile = {
  id: string;
  file_name: string;
  mime_type: string;
  byte_size: number;
  sha256: string;
};

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(API_BASE + path, {
    ...init,
    headers: init?.body instanceof FormData ? init.headers : { "Content-Type": "application/json", ...init?.headers }
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error ?? response.statusText);
  }
  return response.json() as Promise<T>;
}

async function listRequest<T>(path: string): Promise<T[]> {
  const result = await request<T[] | null>(path);
  return Array.isArray(result) ? result : [];
}

export const api = {
  listProjects: () => listRequest<Project>("/api/projects"),
  createProject: (body: { name: string; description: string; domain: string }) =>
    request<Project>("/api/projects", { method: "POST", body: JSON.stringify(body) }),
  listSources: (projectId: string) => listRequest<Source>(`/api/projects/${projectId}/sources`),
  uploadSource: (projectId: string, file: File) => {
    const form = new FormData();
    form.append("file", file);
    return request<Source>(`/api/projects/${projectId}/sources`, { method: "POST", body: form });
  },
  startRun: (sourceId: string) => request<Run>(`/api/sources/${sourceId}/runs`, { method: "POST" }),
  listRuns: (projectId: string) => listRequest<Run>(`/api/projects/${projectId}/runs`),
  listJobs: (projectId: string) => listRequest<Job>(`/api/projects/${projectId}/jobs`),
  listSamples: (projectId: string, status = "") =>
    listRequest<Sample>(`/api/projects/${projectId}/samples?limit=80${status ? `&status=${status}` : ""}`),
  editSample: (sampleId: string, body: { input_text: string; output_text: string; change_reason: string; status: string }) =>
    request<Sample>(`/api/samples/${sampleId}/edit`, {
      method: "PATCH",
      body: JSON.stringify({ ...body, edited_by: "workbench" })
    }),
  listSampleVersions: (sampleId: string) => listRequest<SampleVersion>(`/api/samples/${sampleId}/versions`),
  reviewSample: (sampleId: string, status: string) =>
    request<Sample>(`/api/samples/${sampleId}/review`, {
      method: "PATCH",
      body: JSON.stringify({ status, reviewer: "workbench" })
    }),
  listQualityIssues: (projectId: string) => listRequest<QualityIssue>(`/api/projects/${projectId}/quality-issues`),
  listVersions: (projectId: string) => listRequest<DatasetVersion>(`/api/projects/${projectId}/dataset-versions`),
  listVersionFiles: (versionId: string) => listRequest<DatasetVersionFile>(`/api/dataset-versions/${versionId}/files`),
  versionFileDownloadUrl: (fileId: string) => `${API_BASE}/api/dataset-version-files/${fileId}/download`,
  createVersion: (projectId: string, runId: string, versionName: string) =>
    request<DatasetVersion>(`/api/projects/${projectId}/dataset-versions`, {
      method: "POST",
      body: JSON.stringify({ run_id: runId, version_name: versionName })
    })
};
