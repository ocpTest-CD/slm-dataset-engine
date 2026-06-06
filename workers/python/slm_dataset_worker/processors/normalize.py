from typing import Any


def normalize_record(record: dict[str, Any]) -> dict[str, Any]:
    input_text = first_text(record, ["input", "prompt", "question", "instruction", "content", "text"])
    output_text = first_text(record, ["output", "answer", "response", "completion", "target"])
    if not output_text and "messages" in record:
        input_text, output_text = normalize_messages(record["messages"])
    return {
        "input_text": input_text.strip(),
        "output_text": output_text.strip(),
        "raw_record": record,
    }


def first_text(record: dict[str, Any], keys: list[str]) -> str:
    for key in keys:
        value = record.get(key)
        if isinstance(value, str):
            return value
    return ""


def normalize_messages(messages: object) -> tuple[str, str]:
    if not isinstance(messages, list):
        return "", ""
    user_parts: list[str] = []
    assistant_parts: list[str] = []
    for message in messages:
        if not isinstance(message, dict):
            continue
        role = message.get("role")
        content = message.get("content")
        if not isinstance(content, str):
            continue
        if role in {"user", "system"}:
            user_parts.append(content)
        elif role == "assistant":
            assistant_parts.append(content)
    return "\n".join(user_parts), "\n".join(assistant_parts)

