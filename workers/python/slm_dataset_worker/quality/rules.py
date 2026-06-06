import hashlib


def content_hash(input_text: str, output_text: str) -> str:
    digest = hashlib.sha256()
    digest.update(input_text.encode("utf-8"))
    digest.update(b"\n---\n")
    digest.update(output_text.encode("utf-8"))
    return digest.hexdigest()


def token_count(input_text: str, output_text: str) -> int:
    return len((input_text + " " + output_text).split())


def evaluate(input_text: str, output_text: str, seen_hashes: set[str]) -> tuple[float, list[dict[str, str]], str, int]:
    sample_hash = content_hash(input_text, output_text)
    tokens = token_count(input_text, output_text)
    issues: list[dict[str, str]] = []
    score = 100.0

    if not input_text:
        issues.append(issue("missing_input", "error", "样本缺少输入字段"))
        score -= 40
    if not output_text:
        issues.append(issue("missing_output", "warning", "样本缺少输出字段"))
        score -= 20
    if tokens > 4096:
        issues.append(issue("too_long", "warning", "样本文本过长"))
        score -= 15
    if sample_hash in seen_hashes:
        issues.append(issue("duplicate", "warning", "当前导入批次中发现重复样本"))
        score -= 20

    seen_hashes.add(sample_hash)
    return max(score, 0.0), issues, sample_hash, tokens


def issue(issue_type: str, severity: str, message: str) -> dict[str, str]:
    return {"issue_type": issue_type, "severity": severity, "message": message}

