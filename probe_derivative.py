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
    'messages': [{'role': 'user', 'content': 'What is the derivative of x^3 + 5x^2 - 7x + 12?'}]
}

try:
    req = urllib.request.Request(url, data=json.dumps(payload).encode('utf-8'), headers=headers, method='POST')
    resp = urllib.request.urlopen(req, timeout=90.0)
    print("STATUS:", resp.status)
    data = json.loads(resp.read().decode('utf-8'))
    print("AI RESPONSE:")
    print(data['choices'][0]['message']['content'])
    print("\nSETTLEMENT:")
    print(json.dumps(data.get('settlement', {}), indent=2))
    print("\nUSAGE:")
    print(json.dumps(data.get('usage', {}), indent=2))
except urllib.error.HTTPError as e:
    print("HTTP ERROR:", e.code)
    print("BODY:", e.read().decode('utf-8'))
except Exception as e:
    print("ERR:", type(e), e)
    traceback.print_exc()
