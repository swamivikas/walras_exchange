import urllib.request
import urllib.error
import json
import traceback

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

try:
    req = urllib.request.Request(url, data=json.dumps(payload).encode('utf-8'), headers=headers, method='POST')
    resp = urllib.request.urlopen(req, timeout=35.0)
    print("STATUS:", resp.status)
    print("BODY:", resp.read().decode('utf-8'))
except urllib.error.HTTPError as e:
    print("HTTP ERROR:", e.code)
    print("BODY:", e.read().decode('utf-8'))
except Exception as e:
    print("ERR:", type(e), e)
    traceback.print_exc()
