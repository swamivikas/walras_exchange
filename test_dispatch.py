import urllib.request
import urllib.error
import json
import time

url = 'http://localhost:8080/v1/chat/completions'
headers = {
    'Authorization': 'Bearer walras_live_org_5987bfec_e2ddbec4b678f307a6de07e39f0278d3',
    'Content-Type': 'application/json'
}
payload = {
    'agent_type': 'MATH_SOLVER',
    'model': 'deepseek-r1',
    'messages': [{'role': 'user', 'content': 'Calculate 25 * 4'}]
}

print("Dispatching compute task to Walras Gateway...")
start = time.time()
req = urllib.request.Request(url, data=json.dumps(payload).encode('utf-8'), headers=headers, method='POST')

try:
    with urllib.request.urlopen(req, timeout=30.0) as resp:
        elapsed = time.time() - start
        print(f"✅ Success! Status: {resp.status} (Elapsed: {elapsed:.2f}s)")
        print(resp.read().decode('utf-8'))
except urllib.error.HTTPError as e:
    elapsed = time.time() - start
    print(f"❌ HTTP Error {e.code} after {elapsed:.2f}s:")
    print(e.read().decode('utf-8'))
except Exception as e:
    elapsed = time.time() - start
    print(f"❌ Exception after {elapsed:.2f}s: {type(e)}: {e}")
