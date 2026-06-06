#!/usr/bin/env python3
import json
import os
import tempfile
import time
import urllib.error
import urllib.request
import zipfile
from pathlib import Path
from typing import Any


BASE_URL = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
TIMEOUT_SECONDS = int(os.environ.get("SMOKE_TIMEOUT_SECONDS", "180"))


def request(method: str, path: str, body: Any | None = None, headers: dict[str, str] | None = None) -> Any:
    data = None
    req_headers = headers or {}
    if body is not None:
        if isinstance(body, bytes):
            data = body
        else:
            data = json.dumps(body, ensure_ascii=False).encode("utf-8")
            req_headers = {"Content-Type": "application/json", **req_headers}
    req = urllib.request.Request(BASE_URL + path, data=data, method=method, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} failed: {exc.code} {detail}") from exc
    if not raw:
        return None
    return json.loads(raw.decode("utf-8"))


def upload_file(project_id: str, path: Path) -> dict[str, Any]:
    boundary = f"slm-smoke-{int(time.time() * 1000)}"
    content = path.read_bytes()
    body = b"".join(
        [
            f"--{boundary}\r\n".encode(),
            f'Content-Disposition: form-data; name="file"; filename="{path.name}"\r\n'.encode(),
            b"Content-Type: application/octet-stream\r\n\r\n",
            content,
            b"\r\n",
            f"--{boundary}--\r\n".encode(),
        ]
    )
    return request("POST", f"/api/projects/{project_id}/sources", body, {"Content-Type": f"multipart/form-data; boundary={boundary}"})


def poll_until(label: str, fn, predicate):
    deadline = time.time() + TIMEOUT_SECONDS
    last_value = None
    while time.time() < deadline:
        last_value = fn()
        if predicate(last_value):
            return last_value
        time.sleep(2)
    raise RuntimeError(f"{label} timeout, last={last_value}")


def main() -> None:
    stamp = time.strftime("%Y%m%d-%H%M%S")
    project = request(
        "POST",
        "/api/projects",
        {"name": f"smoke-rag-{stamp}", "description": "端到端烟测项目", "domain": "rag"},
    )
    project_id = project["id"]
    print(f"project={project_id}")

    with tempfile.TemporaryDirectory() as tmp:
        dataset_path = Path(tmp) / "smoke.jsonl"
        records = [
            {
                "input": "什么是 RAG 数据集导出包？",
                "output": "RAG 数据集导出包应包含 chunks.jsonl、documents.jsonl、manifest.json 和质量报告，便于检索系统直接导入。",
            },
            {
                "input": "为什么样本审核需要人工编辑？",
                "output": "人工编辑可以修复格式、补齐答案、删除噪声，并把修改记录保存在样本版本中。",
            },
        ]
        dataset_path.write_text("\n".join(json.dumps(row, ensure_ascii=False) for row in records), encoding="utf-8")

        source = upload_file(project_id, dataset_path)
        run = request("POST", f"/api/sources/{source['id']}/runs")
        run_id = run["id"]
        print(f"source={source['id']} run={run_id}")

        runs = poll_until(
            "import run",
            lambda: request("GET", f"/api/projects/{project_id}/runs"),
            lambda rows: rows and rows[0]["status"] in {"waiting_review", "completed"},
        )
        print(f"import_status={runs[0]['status']} samples={runs[0]['total_samples']}")

        samples = request("GET", f"/api/projects/{project_id}/samples?limit=20")
        if not samples:
            raise RuntimeError("no samples generated")
        first = samples[0]
        request(
            "PATCH",
            f"/api/samples/{first['id']}/edit",
            {
                "input_text": first["input_text"],
                "output_text": first["output_text"] + "\n已通过烟测人工修订。",
                "edited_by": "smoke",
                "change_reason": "端到端烟测编辑",
                "status": "accepted",
            },
        )
        for sample in samples[1:]:
            request("PATCH", f"/api/samples/{sample['id']}/review", {"status": "accepted", "reviewer": "smoke"})
        print(f"accepted_samples={len(samples)}")

        version = request("POST", f"/api/projects/{project_id}/dataset-versions", {"run_id": run_id, "version_name": f"rag-smoke-{stamp}"})
        version_id = version["id"]
        versions = poll_until(
            "export version",
            lambda: request("GET", f"/api/projects/{project_id}/dataset-versions"),
            lambda rows: any(row["id"] == version_id and row["status"] == "ready" for row in rows),
        )
        ready = next(row for row in versions if row["id"] == version_id)
        print(f"version={version_id} status={ready['status']} sample_count={ready['sample_count']}")

        files = request("GET", f"/api/dataset-versions/{version_id}/files")
        zip_file = next((item for item in files if item["file_name"] == "rag-dataset.zip"), None)
        if zip_file is None:
            raise RuntimeError(f"rag-dataset.zip not found: {files}")

        zip_path = Path(tmp) / "rag-dataset.zip"
        with urllib.request.urlopen(BASE_URL + f"/api/dataset-version-files/{zip_file['id']}/download", timeout=30) as resp:
            zip_path.write_bytes(resp.read())
        with zipfile.ZipFile(zip_path) as archive:
            names = set(archive.namelist())
        required = {"chunks.jsonl", "documents.jsonl", "dataset.jsonl", "manifest.json", "quality_report.json", "README.md"}
        missing = required - names
        if missing:
            raise RuntimeError(f"zip missing files: {sorted(missing)}")
        print("smoke=ok")


if __name__ == "__main__":
    main()
