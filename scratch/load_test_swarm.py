import concurrent.futures
import json
import time
import urllib.request
import sys

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
if hasattr(sys.stderr, 'reconfigure'):
    sys.stderr.reconfigure(encoding='utf-8', errors='replace')

URL = "http://localhost:8080/v1/chat/completions"
HEADERS = {
    "Authorization": "Bearer walras_live_org_b44e2188_d7ea75e39247706df4be7c4f36994fd3",
    "Content-Type": "application/json"
}

JOBS = [
    {
        "id": "JOB-1-CALCULUS",
        "agent_type": "MATH_SOLVER",
        "model": "",
        "prompt": "Find the critical points, local extrema, and inflection points of f(x) = x^4 - 4x^3 + 6x^2 - 12 using first and second derivatives. Show step-by-step calculus reasoning."
    },
    {
        "id": "JOB-2-EIGENVALUES",
        "agent_type": "MATH_SOLVER",
        "model": "",
        "prompt": "Determine if the 2x2 matrix A = [[3, -2], [1, 0]] is diagonalizable. Find its eigenvalues, eigenvectors, and write the diagonal matrix D."
    },
    {
        "id": "JOB-3-DIFF-EQUATION",
        "agent_type": "MATH_SOLVER",
        "model": "",
        "prompt": "Solve the second-order ordinary differential equation y'' + 4y' + 4y = 0 with initial conditions y(0) = 2 and y'(0) = -3 using the characteristic equation."
    },
    {
        "id": "JOB-4-LRU-CACHE",
        "agent_type": "CODE_GENERATOR",
        "model": "qwen2.5-coder:7b",
        "prompt": "Implement a thread-safe LRU Cache in Python from scratch using a doubly linked list and hash map. Provide complete O(1) get() and put() methods with clean type annotations."
    },
    {
        "id": "JOB-5-TRIE-ENGINE",
        "agent_type": "CODE_GENERATOR",
        "model": "qwen2.5-coder:7b",
        "prompt": "Write a complete Prefix Tree (Trie) in Python supporting insert(word), search(word), and startsWith(prefix) with full complexity explanations."
    }
]

def dispatch_worker(job):
    job_id = job["id"]
    agent_type = job["agent_type"]
    model = job["model"]
    prompt = job["prompt"]
    
    payload = {
        "agent_type": agent_type,
        "model": model,
        "messages": [{"role": "user", "content": prompt}]
    }
    
    t0 = time.time()
    print(f"🚀 [DISPATCH] {job_id} -> Pool: {agent_type} (Model: {model or 'AUTO'})")
    req = urllib.request.Request(URL, data=json.dumps(payload).encode("utf-8"), headers=HEADERS)
    
    try:
        with urllib.request.urlopen(req, timeout=90.0) as resp:
            elapsed = time.time() - t0
            data = json.loads(resp.read().decode("utf-8"))
            usage = data.get("usage", {})
            settlement = data.get("settlement", {})
            ans = data.get("choices", [{}])[0].get("message", {}).get("content", "")
            
            print(f"✅ [CLEARED] {job_id} in {elapsed:.2f}s | Node: {settlement.get('execution_node_id', 'N/A')[:8]} | Tokens: {usage.get('completion_tokens', 'N/A')} | Cost: {settlement.get('formatted_cost_usd', 'N/A')}")
            return {
                "job_id": job_id,
                "elapsed": elapsed,
                "tokens": usage.get("completion_tokens", 0),
                "cost": settlement.get("formatted_cost_usd", ""),
                "node": settlement.get("execution_node_id", ""),
                "content_preview": ans[:150].replace("\n", " ") + "..."
            }
    except Exception as e:
        elapsed = time.time() - t0
        print(f"❌ [FAILED] {job_id} in {elapsed:.2f}s: {e}")
        return {"job_id": job_id, "error": str(e)}

def main():
    print("=" * 70)
    print("🔥 WALRAS COMPUTE SWARM: FIRING 5 HEAVY PARALLEL JOBS ACROSS GPU NODES")
    print("=" * 70)
    start_all = time.time()
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        results = list(executor.map(dispatch_worker, JOBS))
        
    total_duration = time.time() - start_all
    print("\n" + "=" * 70)
    print(f"🏁 SWARM COMPLETE: 5 Heavy Jobs Processed in {total_duration:.2f}s Total!")
    print("=" * 70)

if __name__ == "__main__":
    main()
