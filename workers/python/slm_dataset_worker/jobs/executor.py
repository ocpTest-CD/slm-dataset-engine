import json
import hashlib
from pathlib import Path
from typing import Any, Callable

import psycopg

from slm_dataset_worker.exporters.jsonl import write_json, write_jsonl
from slm_dataset_worker.parsers.dataset import parse_dataset
from slm_dataset_worker.processors.normalize import normalize_record
from slm_dataset_worker.quality.rules import evaluate


class Executor:
    def __init__(self, database_url: str) -> None:
        self.database_url = database_url

    def execute(
        self,
        job_type: str,
        payload: dict[str, Any],
        progress: Callable[[str, int, str, dict[str, Any] | None], None] | None = None,
    ) -> dict[str, Any]:
        if job_type == "import_dataset":
            return self.import_dataset(payload, progress)
        if job_type == "export_dataset":
            return self.export_dataset(payload, progress)
        if job_type == "mcp_tool_invocation":
            return self.invoke_mcp_tool(payload, progress)
        raise ValueError(f"unsupported job type: {job_type}")

    def import_dataset(self, payload: dict[str, Any], progress: Callable[[str, int, str, dict[str, Any] | None], None] | None = None) -> dict[str, Any]:
        project_id = payload["project_id"]
        run_id = payload["run_id"]
        source_id = payload["source_id"]
        artifact_path = payload["artifact_path"]

        seen_hashes: set[str] = set()
        sample_count = 0
        issue_count = 0
        accepted = 0
        rejected = 0

        with psycopg.connect(self.database_url) as conn:
            conn.execute("UPDATE runs SET status = 'running', started_at = COALESCE(started_at, now()), updated_at = now() WHERE id = %s", (run_id,))
            if progress:
                progress("parse_source", 15, "开始解析数据源", {"source_id": source_id})
            for record in parse_dataset(artifact_path):
                normalized = normalize_record(record)
                input_text = normalized["input_text"]
                output_text = normalized["output_text"]
                score, issues, sample_hash, tokens = evaluate(input_text, output_text, seen_hashes)
                status = "pending_review" if score >= 60 else "rejected"
                if status == "rejected":
                    rejected += 1

                sample_id = conn.execute(
                    """
                    INSERT INTO samples (
                        project_id, run_id, source_id, status, input_text, output_text,
                        raw_record, normalized_record, quality_score, token_count, content_hash
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s::jsonb, %s::jsonb, %s, %s, %s)
                    RETURNING id
                    """,
                    (
                        project_id,
                        run_id,
                        source_id,
                        status,
                        input_text,
                        output_text,
                        json.dumps(record, ensure_ascii=False),
                        json.dumps(normalized, ensure_ascii=False),
                        score,
                        tokens,
                        sample_hash,
                    ),
                ).fetchone()[0]
                sample_count += 1

                for issue in issues:
                    conn.execute(
                        """
                        INSERT INTO quality_issues (
                            project_id, run_id, sample_id, issue_type, severity, message
                        )
                        VALUES (%s, %s, %s, %s, %s, %s)
                        """,
                        (project_id, run_id, sample_id, issue["issue_type"], issue["severity"], issue["message"]),
                    )
                    issue_count += 1

                conn.execute(
                    """
                    INSERT INTO lineage_events (project_id, run_id, sample_id, event_type, actor, metadata)
                    VALUES (%s, %s, %s, 'imported', 'python-worker', %s::jsonb)
                    """,
                    (project_id, run_id, sample_id, json.dumps({"source_id": source_id}, ensure_ascii=False)),
                )

            if progress:
                progress("waiting_review", 100, "导入完成，等待样本审核", {"sample_count": sample_count, "issue_count": issue_count})
            conn.execute(
                """
                UPDATE runs
                SET status = 'waiting_review', progress = 100, total_samples = %s,
                    accepted_samples = %s, rejected_samples = %s, issue_count = %s,
                    finished_at = now(), updated_at = now()
                WHERE id = %s
                """,
                (sample_count, accepted, rejected, issue_count, run_id),
            )

        return {
            "sample_count": sample_count,
            "issue_count": issue_count,
            "status": "waiting_review",
        }

    def export_dataset(self, payload: dict[str, Any], progress: Callable[[str, int, str, dict[str, Any] | None], None] | None = None) -> dict[str, Any]:
        project_id = payload["project_id"]
        run_id = payload.get("run_id") or ""
        version_id = payload["dataset_version_id"]
        export_root = Path(payload["export_path"])
        if export_root.name == "pending":
            export_root = export_root.parent / version_id
        export_root.mkdir(parents=True, exist_ok=True)
        if progress:
            progress("export_dataset", 20, "开始导出数据集", {"dataset_version_id": version_id})

        rows: list[dict[str, Any]] = []
        with psycopg.connect(self.database_url) as conn:
            query = """
                SELECT id, input_text, output_text, quality_score, token_count
                FROM samples
                WHERE project_id = %s AND status = 'accepted'
            """
            args: list[Any] = [project_id]
            if run_id:
                query += " AND run_id = %s"
                args.append(run_id)
            query += " ORDER BY created_at"

            for sample_id, input_text, output_text, quality_score, tokens in conn.execute(query, args):
                rows.append(
                    {
                        "id": str(sample_id),
                        "input": input_text,
                        "output": output_text,
                        "quality_score": float(quality_score or 0),
                        "token_count": tokens,
                    }
                )

            data_path = export_root / "dataset.jsonl"
            manifest_path = export_root / "manifest.json"
            report_path = export_root / "quality_report.json"
            write_jsonl(data_path, rows)

            manifest = {
                "dataset_version_id": version_id,
                "project_id": project_id,
                "run_id": run_id,
                "sample_count": len(rows),
                "files": ["dataset.jsonl", "manifest.json", "quality_report.json"],
            }
            report = {
                "sample_count": len(rows),
                "status": "ready",
                "accepted_only": True,
            }
            write_json(manifest_path, manifest)
            write_json(report_path, report)
            if progress:
                progress("write_artifacts", 80, "导出文件已写入", {"sample_count": len(rows)})

            conn.execute(
                """
                UPDATE dataset_versions
                SET status = 'ready', manifest = %s::jsonb, artifact_path = %s,
                    sample_count = %s, updated_at = now()
                WHERE id = %s
                """,
                (json.dumps(manifest, ensure_ascii=False), str(export_root), len(rows), version_id),
            )
            if run_id:
                conn.execute(
                    """
                    UPDATE runs
                    SET status = 'completed', accepted_samples = %s, updated_at = now()
                    WHERE id = %s
                    """,
                    (len(rows), run_id),
                )

        return {"sample_count": len(rows), "artifact_path": str(export_root), "status": "ready"}

    def invoke_mcp_tool(self, payload: dict[str, Any], progress: Callable[[str, int, str, dict[str, Any] | None], None] | None = None) -> dict[str, Any]:
        workspace_id = payload["workspace_id"]
        project_id = payload["project_id"]
        invocation_id = payload["invocation_id"]
        job_id = payload.get("job_id", "")
        tool_name = payload["tool_name"]
        raw_input = payload.get("input") or "{}"
        input_data = json.loads(raw_input)
        artifact_root = Path(payload["artifact_path"])
        if artifact_root.name == "pending":
            artifact_root = artifact_root.parent / invocation_id
        artifact_root.mkdir(parents=True, exist_ok=True)

        if progress:
            progress("invoke_tool", 25, "开始执行 MCP Tool", {"tool_name": tool_name})

        output = {
            "tool_name": tool_name,
            "input": input_data,
            "message": "MCP Tool 已执行，产物已生成。",
            "artifact_format": "mcp.artifact.v1",
        }
        result_path = artifact_root / "result.json"
        readme_path = artifact_root / "README.md"
        write_json(result_path, output)
        readme_path.write_text(
            "# MCP Tool 调用产物\n\n"
            f"- Tool: {tool_name}\n"
            f"- Invocation: {invocation_id}\n"
            "- result.json 保存本次工具调用输入和输出。\n",
            encoding="utf-8",
        )

        files = [
            self._file_record(result_path, "application/json"),
            self._file_record(readme_path, "text/markdown; charset=utf-8"),
        ]
        manifest = {
            "schema_version": "mcp.artifact.v1",
            "workspace_id": workspace_id,
            "project_id": project_id,
            "invocation_id": invocation_id,
            "tool_name": tool_name,
            "files": [{"name": item["name"], "sha256": item["sha256"], "bytes": item["bytes"]} for item in files],
        }
        manifest_path = artifact_root / "manifest.json"
        write_json(manifest_path, manifest)
        files.append(self._file_record(manifest_path, "application/json"))

        if progress:
            progress("register_artifact", 70, "登记 MCP 调用产物", {"file_count": len(files)})

        with psycopg.connect(self.database_url) as conn:
            artifact_id = conn.execute(
                """
                INSERT INTO artifacts (workspace_id, project_id, invocation_id, job_id, name, artifact_type, status, manifest)
                VALUES (%s, %s, %s, NULLIF(%s, '')::uuid, %s, 'mcp_tool_result', 'ready', %s::jsonb)
                RETURNING id
                """,
                (workspace_id, project_id, invocation_id, job_id, f"{tool_name}-artifact", json.dumps(manifest, ensure_ascii=False)),
            ).fetchone()[0]
            for item in files:
                conn.execute(
                    """
                    INSERT INTO artifact_files (workspace_id, artifact_id, file_name, file_path, mime_type, byte_size, sha256)
                    VALUES (%s, %s, %s, %s, %s, %s, %s)
                    """,
                    (workspace_id, artifact_id, item["name"], item["path"], item["mime_type"], item["bytes"], item["sha256"]),
                )
            conn.execute(
                """
                UPDATE tool_invocations
                SET status = 'succeeded', output = %s::jsonb, duration_ms = 0, updated_at = now()
                WHERE id = %s
                """,
                (json.dumps({"artifact_id": str(artifact_id), "file_count": len(files)}, ensure_ascii=False), invocation_id),
            )
        if progress:
            progress("completed", 100, "MCP Tool 调用完成", {"file_count": len(files)})
        return {"artifact_path": str(artifact_root), "file_count": len(files), "status": "ready"}

    def _file_record(self, path: Path, mime_type: str) -> dict[str, Any]:
        data = path.read_bytes()
        return {
            "name": path.name,
            "path": str(path),
            "mime_type": mime_type,
            "bytes": len(data),
            "sha256": hashlib.sha256(data).hexdigest(),
        }
