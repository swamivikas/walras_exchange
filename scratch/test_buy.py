import urllib.request
import json
import time

url = "http://localhost:8080/v1/chat/completions"
headers = {
    "Authorization": "Bearer walras_live_org_5987bfec_e2ddbec4b678f307a6de07e39f0278d3",
    "Content-Type": "application/json"
}
payload = {
    "agent_type": "MATH_SOLVER",
    "messages": [{"role": "user", "content": "What is 2 + 2?"}]
}

print(f"Sending test job to {url}...")
t0 = time.time()
req = urllib.request.Request(url, data=json.dumps(payload).encode("utf-8"), headers=headers)
try:
    with urllib.request.urlopen(req, timeout=15) as resp:
        duration = time.time() - t0
        print(f"Received response in {duration:.2f}s:")
        print(resp.read().decode("utf-8"))
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.read().decode('utf-8')}")
except Exception as e:
    print(f"Error: {e}")
