import hashlib
import json
import urllib.request

# We can query MySQL or user agents endpoint
for owner_prefix in ["99ea14fd", "8ecf968b"]:
    try:
        # Query owner's agents via internal/public API if available or check keys
        req = urllib.request.Request(f"http://localhost:8080/v1/user/agents?owner_id={owner_prefix}")
        with urllib.request.urlopen(req) as resp:
            data = json.loads(resp.read().decode())
            print(f"Owner {owner_prefix}:", json.dumps(data, indent=2))
    except Exception as e:
        print(f"Error checking {owner_prefix}:", e)
