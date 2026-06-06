import logging
import time

from slm_dataset_worker.config import Config
from slm_dataset_worker.jobs.client import APIClient
from slm_dataset_worker.jobs.executor import Executor


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    cfg = Config()
    client = APIClient(cfg.api_base_url, cfg.worker_id)
    executor = Executor(cfg.database_url)

    logging.info("worker started id=%s api=%s", cfg.worker_id, cfg.api_base_url)
    while True:
        job = client.claim_job()
        if job is None:
            time.sleep(cfg.poll_interval_seconds)
            continue

        logging.info("claimed job id=%s type=%s", job.id, job.job_type)
        try:
            client.mark_running(job.id)
            result = executor.execute(job.job_type, job.payload)
            client.complete(job.id, result)
            logging.info("completed job id=%s", job.id)
        except Exception as exc:  # noqa: BLE001 - worker must report all failures to API
            logging.exception("job failed id=%s", job.id)
            client.fail(job.id, str(exc))


if __name__ == "__main__":
    main()

