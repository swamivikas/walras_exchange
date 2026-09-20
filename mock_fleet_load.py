#!/usr/bin/env python3
"""
WALRAS EXCHANGE - REALISTIC FLEET & MOCK LOAD ENGINE
Maintains 3-4 online worker nodes and continuously oscillates 2-3 in-flight compute jobs,
generating 20%-40% active load on the War Room NOC dashboard.
"""

import threading
import time
import json
import urllib.request
import websocket
import sys

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
if hasattr(sys.stderr, 'reconfigure'):
    sys.stderr.reconfigure(encoding='utf-8', errors='replace')

GATEWAY_WS = "ws://localhost:8080/v1/provider/tunnel"
GATEWAY_HTTP = "http://localhost:8080/v1/chat/completions"
BUYER_KEY = "walras_live_org_5987bfec_e2ddbec4b678f307a6de07e39f0278d3"

# Node 1: MATH_SOLVER
NODE_MATH_KEY = "walras_live_org_29c249b6_e6bf9aa4ef4b4a5eda58bb0df8534ac1"
# Node 2: CODE_GENERATOR
NODE_CODE_KEY = "walras_live_org_5b7a6727_e846263b4e66985dbcd3edaab3bd4565"
# Node 3: OCR_DOCUMENT_PARSER
NODE_OCR_KEY = "walras_live_org_5b7a6727_9393f9e4cf22915b282cba6c45e9471c"

def run_worker_node(name, api_key, domain_type):
    """Runs a simulated high-performance worker daemon on localhost."""
    headers = [f"Authorization: Bearer {api_key}"]

    def on_open(ws):
        print(f"🚀 [FLEET] Node '{name}' ({domain_type}) connected to Gateway tunnel!")

    def on_message(ws, message):
        try:
            data = json.loads(message)
            req_id = data.get("request_id", "req-1")
            prompt = data.get("prompt", "")
            
            # Simulate realistic processing duration (2.8s - 3.8s)
            sim_time = 3.2
            time.sleep(sim_time)
            
            if domain_type == "MATH_SOLVER":
                ans = "### [" + name + " Symbolic Engine]\nCalculated Solution: Verified proof convergence in 2.4s."
            elif domain_type == "CODE_GENERATOR":
                ans = "```typescript\n// Synthesized by " + name + "\nexport function executeTask() {\n  return { status: 'OPTIMAL', latency_ms: 2400 };\n}\n```"
            else:
                ans = "### [" + name + " OCR Extractor]\nStructured Output: Extracted 28 tabular rows with 99.9% token confidence."
            
            payload = {
                "request_id": req_id,
                "response": ans,
                "status": "COMPLETED",
                "prompt_tokens": len(prompt) // 4 + 8,
                "completion_tokens": len(ans) // 4 + 18
            }
            ws.send(json.dumps(payload))
        except Exception as e:
            pass

    def on_error(ws, err):
        pass

    def on_close(ws, code, msg):
        pass

    while True:
        try:
            ws = websocket.WebSocketApp(
                GATEWAY_WS,
                header=headers,
                on_open=on_open,
                on_message=on_message,
                on_error=on_error,
                on_close=on_close
            )
            ws.run_forever(ping_interval=20, ping_timeout=10)
        except Exception:
            pass
        time.sleep(2)


def run_continuous_traffic_generator():
    """Generates rhythmic, staggered compute queries to pin 20%-40% utilization."""
    print("⚡ [TRAFFIC GENERATOR] Starting rhythmic load generator to maintain 20%-40% utilization...")
    
    prompts = [
        {"agent_type": "CODE_GENERATOR", "model": "", "content": "Write an optimized LRU cache with O(1) eviction in Go."},
        {"agent_type": "OCR_DOCUMENT_PARSER", "model": "", "content": "Extract structured JSON schema from invoice INV-2026-9081."},
        {"agent_type": "MATH_SOLVER", "model": "deepseek-r1", "content": "Calculate the Riemann zeta function zeros approximation."},
        {"agent_type": "CODE_GENERATOR", "model": "", "content": "Implement a distributed rate limiter using token bucket algorithm."}
    ]

    p_idx = 0
    while True:
        try:
            item = prompts[p_idx % len(prompts)]
            p_idx += 1

            headers = {
                "Authorization": f"Bearer {BUYER_KEY}",
                "Content-Type": "application/json"
            }
            body = {
                "agent_type": item["agent_type"],
                "model": item["model"],
                "messages": [{"role": "user", "content": item["content"]}]
            }

            # Fire request in background thread so multiple can overlap and create in-flight spikes
            def dispatch(b, h):
                try:
                    req = urllib.request.Request(GATEWAY_HTTP, data=json.dumps(b).encode('utf-8'), headers=h, method='POST')
                    with urllib.request.urlopen(req, timeout=15.0) as resp:
                        pass
                except Exception:
                    pass

            threading.Thread(target=dispatch, args=(body, headers), daemon=True).start()

        except Exception:
            pass

        # Stagger every 0.8 - 1.0 seconds so 3-4 jobs are consistently overlapping in flight (25% - 35% load)
        time.sleep(0.9)


if __name__ == "__main__":
    print("=================================================================")
    print("🔥 WALRAS WAR ROOM - MOCK FLEET & ACTIVE LOAD INJECTOR")
    print("=================================================================")
    
    # 1. Start Math Solver Worker
    t1 = threading.Thread(target=run_worker_node, args=("DeepSeek-SymbolicRig-01", NODE_MATH_KEY, "MATH_SOLVER"), daemon=True)
    t1.start()

    # 2. Start Code Generator Worker
    t2 = threading.Thread(target=run_worker_node, args=("Claude-3.5-CodeRig-02", NODE_CODE_KEY, "CODE_GENERATOR"), daemon=True)
    t2.start()

    # 3. Start OCR Document Parser Worker
    t3 = threading.Thread(target=run_worker_node, args=("Vision-OCR-Rig-03", NODE_OCR_KEY, "OCR_DOCUMENT_PARSER"), daemon=True)
    t3.start()

    # Wait 2 seconds for worker registrations
    time.sleep(2)

    # 4. Start Rhythmic Traffic Generator
    run_continuous_traffic_generator()
