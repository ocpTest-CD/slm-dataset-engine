import json
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass
class Job:
    id: str
    project_id: str
    run_id: str
    job_type: str
    payload: dict[str, Any]


class APIClient:
    def __init__(self, base_url: str, worker_id: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.worker_id = worker_id

    def claim_job(self) -> Job | None:
        response = self._request("POST", "/api/jobs/claim", {"worker_id": self.worker_id})
        if response is None:
            return None
        payload = json.loads(response["payload"])
        return Job(
            id=response["id"],
            project_id=response.get("project_id", ""),
            run_id=response.get("run_id", ""),
            job_type=response["job_type"],
            payload=payload,
        )

    def mark_running(self, job_id: str) -> None:
        self._request("PATCH", f"/api/jobs/{job_id}/running", {})

    def heartbeat(self, job_id: str) -> None:
        self._request("PATCH", f"/api/jobs/{job_id}/heartbeat", {})

    def complete(self, job_id: str, result: dict[str, Any]) -> None:
        self._request("PATCH", f"/api/jobs/{job_id}/complete", {"result": result})

    def fail(self, job_id: str, message: str) -> None:
        self._request("PATCH", f"/api/jobs/{job_id}/fail", {"error_message": message})

    def _request(self, method: str, path: str, body: dict[str, Any]) -> dict[str, Any] | None:
        data = json.dumps(body).encode("utf-8")
        req = urllib.request.Request(
            self.base_url + path,
            data=data,
            method=method,
            headers={"Content-Type": "application/json"},
        )
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                if resp.status == 204:
                    return None
                raw = resp.read()
        except urllib.error.HTTPError as exc:
            if exc.code == 204:
                return None
            raise
        if not raw:
            return {}
        return json.loads(raw.decode("utf-8"))

