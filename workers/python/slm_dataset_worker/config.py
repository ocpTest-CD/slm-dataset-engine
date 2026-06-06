import os
import socket


class Config:
    def __init__(self) -> None:
        self.api_base_url = os.getenv("API_BASE_URL", "http://127.0.0.1:8080")
        self.database_url = os.getenv(
            "DATABASE_URL",
            "postgres://slm_dataset_engine:slm_dataset_engine@localhost:5432/slm_dataset_engine?sslmode=disable",
        )
        self.worker_id = os.getenv("WORKER_ID", f"worker-{socket.gethostname()}")
        self.poll_interval_seconds = float(os.getenv("POLL_INTERVAL_SECONDS", "2"))

