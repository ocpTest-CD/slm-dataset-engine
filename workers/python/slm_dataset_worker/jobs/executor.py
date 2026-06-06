import json
import hashlib
from pathlib import Path
from typing import Any, Callable
from zipfile import ZIP_DEFLATED, ZipFile

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
            progress("collect_samples", 15, "收集已接受样本", {"dataset_version_id": version_id})

        rows: list[dict[str, Any]] = []
        chunks: list[dict[str, Any]] = []
        documents: dict[str, dict[str, Any]] = {}
        with psycopg.connect(self.database_url) as conn:
            query = """
                SELECT s.id, s.source_id, COALESCE(src.filename, ''), COALESCE(src.source_type, ''),
                    s.input_text, s.output_text, s.quality_score, s.token_count, s.status
                FROM samples s
                LEFT JOIN sources src ON src.id = s.source_id
                WHERE s.project_id = %s AND s.status = 'accepted'
            """
            args: list[Any] = [project_id]
            if run_id:
                query += " AND s.run_id = %s"
                args.append(run_id)
            query += " ORDER BY s.created_at"

            for index, row in enumerate(conn.execute(query, args)):
                sample_id, source_id, source_file, source_type, input_text, output_text, quality_score, tokens, status = row
                text = output_text or input_text
                rows.append(
                    {
                        "id": str(sample_id),
                        "input": input_text,
                        "output": output_text,
                        "quality_score": float(quality_score or 0),
                        "token_count": tokens,
                        "source_id": str(source_id or ""),
                        "source_file": source_file,
                        "status": status,
                    }
                )
                documents.setdefault(
                    str(source_id or "manual"),
                    {"id": str(source_id or "manual"), "source_file": source_file or "manual", "source_type": source_type or "sample", "chunk_count": 0},
                )
                documents[str(source_id or "manual")]["chunk_count"] += 1
                chunks.append(
                    {
                        "id": f"chunk_{sample_id}",
                        "text": text,
                        "metadata": {
                            "project_id": project_id,
                            "dataset_version_id": version_id,
                            "run_id": run_id,
                            "sample_id": str(sample_id),
                            "source_id": str(source_id or ""),
                            "source_file": source_file,
                            "source_type": source_type,
                            "chunk_index": index,
                            "quality_score": float(quality_score or 0),
                            "token_count": tokens,
                        },
                    }
                )
            if progress:
                progress("write_rag_files", 60, "写入 RAG JSONL 和报告", {"chunk_count": len(chunks)})

            data_path = export_root / "dataset.jsonl"
            chunks_path = export_root / "chunks.jsonl"
            documents_path = export_root / "documents.jsonl"
            manifest_path = export_root / "manifest.json"
            report_path = export_root / "quality_report.json"
            readme_path = export_root / "README.md"
            zip_path = export_root / "rag-dataset.zip"
            write_jsonl(data_path, rows)
            write_jsonl(chunks_path, chunks)
            write_jsonl(documents_path, list(documents.values()))

            manifest = {
                "schema_version": "rag.dataset.v1",
                "dataset_version_id": version_id,
                "project_id": project_id,
                "run_id": run_id,
                "sample_count": len(rows),
                "chunk_count": len(chunks),
                "export_type": "rag_zip",
                "files": ["rag-dataset.zip", "chunks.jsonl", "documents.jsonl", "dataset.jsonl", "manifest.json", "quality_report.json", "README.md"],
                "chunking": {"strategy": "sample_as_chunk", "overlap_tokens": 0},
            }
            report = {
                "sample_count": len(rows),
                "chunk_count": len(chunks),
                "status": "ready",
                "accepted_only": True,
            }
            write_json(manifest_path, manifest)
            write_json(report_path, report)
            readme_path.write_text(
                "# RAG 数据集包\n\n"
                "- `chunks.jsonl`：可直接导入 RAG 的 chunk 数据。\n"
                "- `documents.jsonl`：来源文档聚合信息。\n"
                "- `dataset.jsonl`：审核后的样本数据。\n"
                "- `manifest.json`：版本、来源和文件清单。\n",
                encoding="utf-8",
            )
            with ZipFile(zip_path, "w", ZIP_DEFLATED) as archive:
                for path in [chunks_path, documents_path, data_path, manifest_path, report_path, readme_path]:
                    archive.write(path, path.name)
            if progress:
                progress("register_files", 85, "登记可下载数据集文件", {"zip": str(zip_path)})

            conn.execute(
                """
                UPDATE dataset_versions
                SET status = 'ready', manifest = %s::jsonb, artifact_path = %s,
                    sample_count = %s, updated_at = now()
                WHERE id = %s
                """,
                (json.dumps(manifest, ensure_ascii=False), str(export_root), len(rows), version_id),
            )
            conn.execute("DELETE FROM dataset_version_files WHERE dataset_version_id = %s", (version_id,))
            for path, mime_type in [
                (zip_path, "application/zip"),
                (chunks_path, "application/x-ndjson"),
                (documents_path, "application/x-ndjson"),
                (data_path, "application/x-ndjson"),
                (manifest_path, "application/json"),
                (report_path, "application/json"),
                (readme_path, "text/markdown; charset=utf-8"),
            ]:
                file_info = self._file_info(path)
                conn.execute(
                    """
                    INSERT INTO dataset_version_files (dataset_version_id, file_name, file_path, mime_type, byte_size, sha256)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (version_id, path.name, str(path), mime_type, file_info["bytes"], file_info["sha256"]),
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

    def _file_info(self, path: Path) -> dict[str, Any]:
        data = path.read_bytes()
        return {"bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
