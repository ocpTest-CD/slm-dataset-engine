FROM python:3.12-slim

WORKDIR /app
ENV PYTHONUNBUFFERED=1

COPY workers/python/pyproject.toml ./
COPY workers/python/slm_dataset_worker ./slm_dataset_worker
RUN pip install --no-cache-dir .
RUN useradd --uid 1001 --create-home appuser
USER appuser

CMD ["slm-dataset-worker"]
