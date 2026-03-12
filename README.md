# 🌈 llm-prism

**The Privacy Firewall for LLMs.** Stop leaking secrets (API Keys, PII) to AI providers.

`llm-prism` is a local transparent proxy that redacts sensitive information **locally** before it ever leaves your machine.

---

| Feature | 🔴 Direct Connection | 🟢 With llm-prism |
| :--- | :--- | :--- |
| **Data Privacy** | Secrets sent to Cloud | **Redacted Locally** |
| **Provider Sees** | `key: "sk-7d...363e"` | `key: "[REDACTED]"` |
| **Streaming** | Standard | **Real-time filtering** |

---

## 🚀 Quick Start (30s)

### 1. Install
```bash
go install github.com/wangyihang/llm-prism@latest
```

### 2. Setup
```bash
llm-prism sync  # Update redaction rules (Gitleaks compatible)
```

### 3. Run
```bash
export LLM_PRISM_API_KEY=sk-your-real-key
llm-prism run
```

---

## 🛠️ Integration

Connect your favorite tools by changing the API Base URL:

### Claude Code
```bash
export ANTHROPIC_BASE_URL=http://localhost:4000
claude
```

### Cursor / Aider / OpenAI SDK
Simply point your `base_url` to `http://localhost:4000`.

---

## ✨ Features

- **🛡️ Auto-Redaction**: Detects 100+ secret types (AWS, Stripe, OpenAI, etc.).
- **⚡ Zero Latency**: Specialized SSE engine for real-time streaming.
- **🔍 Deep Scan**: Recursively traverses nested JSON (works with Claude's thinking blocks).
- **📊 Local Audit**: Keeps a `llm-prism-detections.jsonl` for your own security review.

---

## 📖 How it works

1. **Intercept**: Sits between your CLI/IDE and the LLM API.
2. **Sanitize**: Scans the request body against Gitleaks-compatible rules.
3. **Redact**: Replaces any matched secrets with `[REDACTED_SECRET]`.
4. **Forward**: Sends the "clean" request to the provider.

---

## License
MIT. See `LICENSE` for details.
