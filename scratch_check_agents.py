import urllib.request
import json

for owner_id in ["5b7a6727-f92a-447b-9f64-0b89b24b62e2", "29c249b6-06fa-401f-8fd4-417c6de3d796"]:
    req = urllib.request.Request(f"http://localhost:8080/v1/user/agents?owner_id={owner_id}")
    try:
        with urllib.request.urlopen(req) as resp:
            data = json.loads(resp.read().decode())
            agents = data.get("agents", [])
            print(f"\nOwner {owner_id[:8]} -> {len(agents)} agents:")
            for a in agents:
                print(f"  • {a.get('agent_id')} | Name: {a.get('name')} | Type: {a.get('agent_type')} | Key: {a.get('api_key_prefix')}")
    except Exception as e:
        print("Err:", e)
