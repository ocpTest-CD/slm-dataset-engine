import csv
import json
from pathlib import Path
from typing import Any, Iterable


def parse_dataset(path: str) -> Iterable[dict[str, Any]]:
    suffix = Path(path).suffix.lower()
    if suffix == ".jsonl":
        yield from parse_jsonl(path)
    elif suffix == ".csv":
        yield from parse_csv(path)
    elif suffix in {".md", ".markdown"}:
        yield from parse_markdown(path)
    else:
        yield from parse_text(path)


def parse_jsonl(path: str) -> Iterable[dict[str, Any]]:
    with open(path, "r", encoding="utf-8") as fh:
        for line_number, line in enumerate(fh, start=1):
            text = line.strip()
            if not text:
                continue
            try:
                record = json.loads(text)
            except json.JSONDecodeError:
                record = {"input": text, "_line": line_number, "_parse_error": "json_decode"}
            yield record


def parse_csv(path: str) -> Iterable[dict[str, Any]]:
    with open(path, "r", encoding="utf-8", newline="") as fh:
        reader = csv.DictReader(fh)
        for row in reader:
            yield dict(row)


def parse_markdown(path: str) -> Iterable[dict[str, Any]]:
    content = Path(path).read_text(encoding="utf-8")
    blocks = [block.strip() for block in content.split("\n\n") if block.strip()]
    for index, block in enumerate(blocks, start=1):
        yield {"input": block, "_block": index}


def parse_text(path: str) -> Iterable[dict[str, Any]]:
    content = Path(path).read_text(encoding="utf-8")
    for index, line in enumerate(content.splitlines(), start=1):
        text = line.strip()
        if text:
            yield {"input": text, "_line": index}

