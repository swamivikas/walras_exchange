# Walras Compute Clearinghouse - MCP Server

Official Model Context Protocol (MCP) server for **Walras Exchange**. Connects Claude Desktop, Cursor IDE, Windsurf, and autonomous agent frameworks directly to decentralized GPU clusters.

---

## ⚡ Features
- **Pure Python, Zero Dependencies**: Runs out of the box on Python 3.8+ using standard library `sys.stdin` and `sys.stdout`. No package conflicts.
- **Domain-Centric Capability Routing**: Hires pools (`MATH_SOLVER`, `CODE_GENERATOR`, `OCR_DOCUMENT_PARSER`, `GENERAL_LLM_INFERENCE`) instead of vendor lock-in.
- **Sub-50ns In-Memory Matching**: Queries are routed through the Walras Go Hot-RAM engine directly to the lowest-latency online node.

---

## 🛠️ Available Tools

| Tool Name | Domain Pool | Description |
| :--- | :--- | :--- |
| **`walras_solve_math`** | `MATH_SOLVER` | High-precision calculus, differential equations, symbolic proofs, linear algebra. |
| **`walras_generate_code`** | `CODE_GENERATOR` | Multi-file code generation, architecture design, and refactoring. |
| **`walras_parse_document`** | `OCR_DOCUMENT_PARSER` | Hardware-accelerated OCR and structured invoice/document parsing. |
| **`walras_run_compute`** | `GENERAL_LLM_INFERENCE` | General LLM reasoning across verified decentralized open-source weights. |
| **`walras_network_status`** | Real-Time Telemetry | Queries active Hot-RAM nodes and available concurrency slots. |

---

## 🚀 Setup & Integration

### Option 1: Claude Desktop
Add this to your Claude Desktop config (`%APPDATA%\Claude\claude_desktop_config.json` on Windows or `~/Library/Application Support/Claude/claude_desktop_config.json` on Mac):

```json
{
  "mcpServers": {
    "walras-compute": {
      "command": "python",
      "args": ["<PATH_TO_REPO>\\mcp-server\\server.py"],
      "env": {
        "WALRAS_API_KEY": "walras_live_org_...",
        "WALRAS_GATEWAY_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Option 2: Cursor IDE
Add this to your `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "walras": {
      "command": "python",
      "args": ["<PATH_TO_REPO>\\mcp-server\\server.py"],
      "env": {
        "WALRAS_API_KEY": "walras_live_org_...",
        "WALRAS_GATEWAY_URL": "http://localhost:8080"
      }
    }
  }
}
```

---

## 🧪 Testing the Server
Run the included test harness to simulate a live MCP host talking over `stdio`:

```bash
python mcp-server/test_mcp_client.py
```
