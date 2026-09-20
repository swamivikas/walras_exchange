#!/usr/bin/env python3
"""
WALRAS EXCHANGE - MODEL CONTEXT PROTOCOL (MCP) SERVER
Pure-Python, Zero-Dependency Standard MCP Host for Claude Desktop, Cursor, and Autonomous AI Agents.

Protocol: JSON-RPC 2.0 over stdio (Specification: 2024-11-05)
Clearinghouse Ingress: Targets Walras Gateway (http://localhost:8080/v1/chat/completions)
"""

import sys
import json
import os
import urllib.request
import urllib.error

# Ensure UTF-8 I/O across platforms
if hasattr(sys.stdin, 'reconfigure'):
    sys.stdin.reconfigure(encoding='utf-8')
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8')

GATEWAY_URL = os.environ.get("WALRAS_GATEWAY_URL", "http://localhost:8080").rstrip("/")
API_KEY = os.environ.get("WALRAS_API_KEY", "walras_live_org_5987bfec_e2ddbec4b678f307a6de07e39f0278d3")

# Domain Capabilities supported by Walras Decentralized Clearinghouse
TOOLS_REGISTRY = [
    {
        "name": "walras_solve_math",
        "description": "Dispatches mathematical calculations, symbolic calculus, differential equations, and proofs to the Walras MATH_SOLVER decentralized GPU pool. Automatically routes to the lowest-latency, highest-capacity verified compute node.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "problem": {
                    "type": "string",
                    "description": "The mathematical problem, proof, or numerical computation to solve."
                },
                "preferred_model": {
                    "type": "string",
                    "description": "Optional: Specific model name if desired; otherwise the clearinghouse auto-matches the best online node in the MATH_SOLVER pool."
                }
            },
            "required": ["problem"]
        }
    },
    {
        "name": "walras_generate_code",
        "description": "Dispatches code generation, multi-file refactoring, and algorithm synthesis to the Walras CODE_GENERATOR decentralized GPU pool.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "specification": {
                    "type": "string",
                    "description": "The programming task, function specification, or refactoring requirements."
                },
                "language": {
                    "type": "string",
                    "description": "Optional: Programming language (e.g., 'Go', 'Python', 'Rust', 'TypeScript')."
                },
                "preferred_model": {
                    "type": "string",
                    "description": "Optional: Preferred model identifier; defaults to auto-routing to the best online node."
                }
            },
            "required": ["specification"]
        }
    },
    {
        "name": "walras_parse_document",
        "description": "Hardware-accelerated OCR, table extraction, and structured multimodal document understanding via the Walras OCR_DOCUMENT_PARSER pool.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "document_content_or_url": {
                    "type": "string",
                    "description": "Text, raw image data URI, or URL of the document/invoice/receipt to parse."
                }
            },
            "required": ["document_content_or_url"]
        }
    },
    {
        "name": "walras_run_compute",
        "description": "Dispatches general AI reasoning and inference tasks across any active Walras domain pool (GENERAL_LLM_INFERENCE, NLP_TRANSLATOR, etc.).",
        "inputSchema": {
            "type": "object",
            "properties": {
                "prompt": {
                    "type": "string",
                    "description": "The prompt or task instructions for the compute node."
                },
                "agent_type": {
                    "type": "string",
                    "description": "Target domain pool (e.g., 'GENERAL_LLM_INFERENCE', 'MATH_SOLVER', 'CODE_GENERATOR'). Defaults to GENERAL_LLM_INFERENCE."
                },
                "preferred_model": {
                    "type": "string",
                    "description": "Optional model identifier."
                }
            },
            "required": ["prompt"]
        }
    },
    {
        "name": "walras_network_status",
        "description": "Queries the live Walras Go Hot-RAM clearinghouse for real-time node count, active concurrency capacity, and pool operational status.",
        "inputSchema": {
            "type": "object",
            "properties": {}
        }
    }
]


def send_response(response_dict):
    """Writes a single JSON-RPC line to stdout followed by newline and flush."""
    json_str = json.dumps(response_dict, ensure_ascii=False)
    sys.stdout.write(json_str + "\n")
    sys.stdout.flush()


def dispatch_to_gateway(agent_type, prompt, model=""):
    """Dispatches a task payload to the Walras Gateway /v1/chat/completions endpoint."""
    url = f"{GATEWAY_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    payload = {
        "agent_type": agent_type,
        "model": model if model else "",
        "messages": [
            {"role": "user", "content": prompt}
        ]
    }

    req_data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=req_data, headers=headers, method="POST")

    try:
        with urllib.request.urlopen(req, timeout=30.0) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            choices = data.get("choices", [])
            if choices and "message" in choices[0]:
                content = choices[0]["message"].get("content", "")
                usage = data.get("usage", {})
                tokens_info = f"\n\n[Walras Clearinghouse Settlement: Model={data.get('model', 'Node')}, Tokens={usage.get('total_tokens', 'N/A')}]"
                return content + tokens_info
            return json.dumps(data)
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8")
        return f"[Walras Gateway Error HTTP {e.code}]: {err_body}"
    except Exception as e:
        return f"[Walras Connection Error]: Failed to connect to {url}: {str(e)}"


def query_network_status():
    """Queries the live Go Hot-RAM telemetry status."""
    # Attempt internal ops port 8090 first, fallback to public health on 8080
    telemetry_url = "http://localhost:8090/v1/ops/telemetry"
    try:
        req = urllib.request.Request(telemetry_url)
        with urllib.request.urlopen(req, timeout=3.0) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            total_nodes = data.get("total_online_nodes", 0)
            capacity = data.get("total_network_capacity", 0)
            in_flight = data.get("total_in_flight_jobs", 0)
            pools = data.get("domain_pools", {})

            pool_summary = []
            for name, stats in pools.items():
                if stats.get("online_nodes", 0) > 0:
                    pool_summary.append(
                        f"  • {name}: {stats.get('online_nodes')} nodes online | "
                        f"{stats.get('available_slots')}/{stats.get('total_capacity')} slots free | "
                        f"In-Flight: {stats.get('in_flight')}"
                    )

            pools_text = "\n".join(pool_summary) if pool_summary else "  • No specialized nodes online"

            return (
                f"### [Walras Decentralized Compute Network Status]\n"
                f"• Registered GPU Nodes in Hot RAM: {total_nodes}\n"
                f"• Network Concurrency Capacity: {capacity} simultaneous slots\n"
                f"• In-Flight Executions: {in_flight}\n\n"
                f"**Active Domain Pools:**\n{pools_text}"
            )
    except Exception:
        # Fallback to public gateway health check
        try:
            health_url = f"{GATEWAY_URL}/health"
            with urllib.request.urlopen(urllib.request.Request(health_url), timeout=2.0) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return f"[Walras Gateway]: Online ({data.get('status', 'ok')}) on {GATEWAY_URL}"
        except Exception as e:
            return f"[Walras Gateway Offline]: Unable to reach {GATEWAY_URL} ({str(e)})"


def handle_tool_call(tool_name, arguments):
    """Executes a tool call and returns an MCP-compliant result content array."""
    if tool_name == "walras_solve_math":
        problem = arguments.get("problem", "")
        preferred_model = arguments.get("preferred_model", "")
        res = dispatch_to_gateway("MATH_SOLVER", problem, preferred_model)
        return [{"type": "text", "text": res}]

    elif tool_name == "walras_generate_code":
        spec = arguments.get("specification", "")
        lang = arguments.get("language", "")
        prompt = f"Language: {lang}\n\nSpecification:\n{spec}" if lang else spec
        preferred_model = arguments.get("preferred_model", "")
        res = dispatch_to_gateway("CODE_GENERATOR", prompt, preferred_model)
        return [{"type": "text", "text": res}]

    elif tool_name == "walras_parse_document":
        doc = arguments.get("document_content_or_url", "")
        res = dispatch_to_gateway("OCR_DOCUMENT_PARSER", doc, "")
        return [{"type": "text", "text": res}]

    elif tool_name == "walras_run_compute":
        prompt = arguments.get("prompt", "")
        agent_type = arguments.get("agent_type", "GENERAL_LLM_INFERENCE")
        preferred_model = arguments.get("preferred_model", "")
        res = dispatch_to_gateway(agent_type, prompt, preferred_model)
        return [{"type": "text", "text": res}]

    elif tool_name == "walras_network_status":
        status_text = query_network_status()
        return [{"type": "text", "text": status_text}]

    else:
        return [{"type": "text", "text": f"Error: Unknown tool '{tool_name}'"}]


def main():
    """Main JSON-RPC stdio event loop."""
    sys.stderr.write("[Walras MCP Server] Process started. Listening on stdio...\n")
    sys.stderr.flush()

    while True:
        line = sys.stdin.readline()
        if not line:
            break

        line = line.strip()
        if not line:
            continue

        try:
            req = json.loads(line)
        except Exception as e:
            sys.stderr.write(f"[Walras MCP Error] Invalid JSON: {e}\n")
            sys.stderr.flush()
            continue

        msg_id = req.get("id")
        method = req.get("method")
        params = req.get("params", {})

        # Handle MCP Handshake
        if method == "initialize":
            send_response({
                "jsonrpc": "2.0",
                "id": msg_id,
                "result": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {
                        "tools": {
                            "listChanged": False
                        }
                    },
                    "serverInfo": {
                        "name": "walras-compute-clearinghouse",
                        "version": "1.0.0"
                    }
                }
            })

        elif method == "notifications/initialized" or method == "initialized":
            # Client notification acknowledging initialization
            pass

        elif method == "tools/list":
            send_response({
                "jsonrpc": "2.0",
                "id": msg_id,
                "result": {
                    "tools": TOOLS_REGISTRY
                }
            })

        elif method == "tools/call":
            tool_name = params.get("name")
            arguments = params.get("arguments", {})
            content = handle_tool_call(tool_name, arguments)
            send_response({
                "jsonrpc": "2.0",
                "id": msg_id,
                "result": {
                    "content": content,
                    "isError": False
                }
            })

        elif method == "ping":
            send_response({
                "jsonrpc": "2.0",
                "id": msg_id,
                "result": {}
            })

        else:
            # Method not recognized
            if msg_id is not None:
                send_response({
                    "jsonrpc": "2.0",
                    "id": msg_id,
                    "error": {
                        "code": -32601,
                        "message": f"Method not found: {method}"
                    }
                })


if __name__ == "__main__":
    main()
