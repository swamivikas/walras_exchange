import urllib.request
import json
import time

url = "http://localhost:8080/v1/chat/completions"
headers = {
    "Authorization": "Bearer walras_live_org_b44e2188_d7ea75e39247706df4be7c4f36994fd3",
    "Content-Type": "application/json"
}
payload = {
    "agent_type": "MATH_SOLVER",
    "model": "deepseek-r1:1.5b",
    "messages": [{"role": "user", "content": "Solve for x: 5*x^2 - 20 = 0 and explain step by step."}]
}

print(f"Sending test job to {url}...")
t0 = time.time()
req = urllib.request.Request(url, data=json.dumps(payload).encode("utf-8"), headers=headers)
try:
    with urllib.request.urlopen(req, timeout=45.0) as resp:
        duration = time.time() - t0
        print(f"Received response in {duration:.2f}s:")
        print(resp.read().decode("utf-8"))
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.read().decode('utf-8')}")
except Exception as e:
    print(f"Error: {e}")
