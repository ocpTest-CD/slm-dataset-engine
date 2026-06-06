import json
from pathlib import Path
from typing import Any

import psycopg

from slm_dataset_worker.exporters.jsonl import write_json, write_jsonl
from slm_dataset_worker.parsers.dataset import parse_dataset
from slm_dataset_worker.processors.normalize import normalize_record
from slm_dataset_worker.quality.rules import evaluate


class Executor:
    def __init__(self, database_url: str) -> None:
        self.database_url = database_url

    def execute(self, job_type: str, payload: dict[str, Any]) -> dict[str, Any]:
        if job_type == "import_dataset":
            return self.import_dataset(payload)
        if job_type == "export_dataset":
            return self.export_dataset(payload)
        raise ValueError(f"unsupported job type: {job_type}")

    def import_dataset(self, payload: dict[str, Any]) -> dict[str, Any]:
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

    def export_dataset(self, payload: dict[str, Any]) -> dict[str, Any]:
        project_id = payload["project_id"]
        run_id = payload.get("run_id") or ""
        version_id = payload["dataset_version_id"]
        export_root = Path(payload["export_path"])
        if export_root.name == "pending":
            export_root = export_root.parent / version_id
        export_root.mkdir(parents=True, exist_ok=True)

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

