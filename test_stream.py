import urllib.request
import json
import time
import sys

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

URL = "http://localhost:8080/v1/chat/completions"
HEADERS = {
    "Authorization": "Bearer walras_live_org_b44e2188_d7ea75e39247706df4be7c4f36994fd3",
    "Content-Type": "application/json"
}

payload = {
    "agent_type": "MATH_SOLVER",
    "model": "deepseek-r1:1.5b",
    "stream": True,
    "messages": [
        {"role": "user", "content": "Calculate the derivative of x^4 - 4x^2 + 7 step by step."}
    ]
}

print("=================================================================")
print("⚡ DISPATCHING STREAMING COMPUTE REQUEST (SSE)...")
print("=================================================================\n")

start_time = time.time()
ttft = None
token_count = 0

req = urllib.request.Request(URL, data=json.dumps(payload).encode('utf-8'), headers=HEADERS, method='POST')

with urllib.request.urlopen(req, timeout=30.0) as resp:
    print(f"📡 Connected! HTTP Status: {resp.status}")
    print("💬 Live Streaming Response from Remote Node:\n-------------------------------------------")

    for raw_line in resp:
        line = raw_line.decode('utf-8', errors='replace').strip()
        if not line:
            continue
        if line.startswith("data: "):
            content = line[6:]
            if content == "[DONE]":
                break
            
            try:
                chunk = json.loads(content)
                delta = chunk.get("choices", [{}])[0].get("delta", {}).get("content", "")
                if delta:
                    if ttft is None:
                        ttft = (time.time() - start_time) * 1000
                    token_count += 1
                    sys.stdout.write(delta)
                    sys.stdout.flush()

                # Print settlement if final chunk
                if chunk.get("settlement"):
                    settlement = chunk["settlement"]
                    print("\n\n=======================================================")
                    print("🌐 [ATOMIC SETTLEMENT RECEIPT (SSE)]")
                    print(f"• Execution Node ID : {settlement.get('execution_node_id')}")
                    print(f"• Cleared Cost      : {settlement.get('formatted_cost_usd')}")
                    print(f"• Platform Fee      : {settlement.get('platform_fee_micro_usd')} µUSD")
                    print("=======================================================")
            except Exception as e:
                pass

total_time = (time.time() - start_time) * 1000
print(f"\n⏱️  PERFORMANCE BENCHMARKS:")
ttft_str = f"{ttft:.1f} ms" if ttft is not None else "N/A"
print(f"• Time-to-First-Token (TTFT): {ttft_str}  (Perceptual local latency!)")
print(f"• Total Streaming Duration  : {total_time:.1f} ms")
print(f"• Chunks Emitted            : {token_count} chunks")
