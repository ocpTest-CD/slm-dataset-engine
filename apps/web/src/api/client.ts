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
  status: string;
  input_text: string;
  output_text: string;
  quality_score: number | null;
  token_count: number;
  created_at: string;
};

export type QualityIssue = {
  id: string;
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
  listSamples: (projectId: string, status = "") =>
    listRequest<Sample>(`/api/projects/${projectId}/samples?limit=80${status ? `&status=${status}` : ""}`),
  reviewSample: (sampleId: string, status: string) =>
    request<Sample>(`/api/samples/${sampleId}/review`, {
      method: "PATCH",
      body: JSON.stringify({ status, reviewer: "workbench" })
    }),
  listQualityIssues: (projectId: string) => listRequest<QualityIssue>(`/api/projects/${projectId}/quality-issues`),
  listVersions: (projectId: string) => listRequest<DatasetVersion>(`/api/projects/${projectId}/dataset-versions`),
  createVersion: (projectId: string, runId: string, versionName: string) =>
    request<DatasetVersion>(`/api/projects/${projectId}/dataset-versions`, {
      method: "POST",
      body: JSON.stringify({ run_id: runId, version_name: versionName })
    })
};
