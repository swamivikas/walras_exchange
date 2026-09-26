import urllib.request
import json

try:
    with urllib.request.urlopen("http://localhost:8090/v1/ops/telemetry") as resp:
        data = json.loads(resp.read().decode())
        print("Total Online Nodes:", data.get("total_online_nodes"))
        print("\nActive Nodes in Pool:")
        for n in data.get("nodes", []):
            print(f"  • AgentID: {n.get('agent_id')} | Name: {n.get('node_name')} | Model: {n.get('model')} | Status: {n.get('status')}")
        print("\nDomain Pools:")
        for pool, stats in data.get("domain_pools", {}).items():
            if stats.get("online_nodes", 0) > 0:
                print(f"  • {pool}: {stats.get('online_nodes')} nodes | slots: {stats.get('available_slots')}/{stats.get('total_capacity')}")
except Exception as e:
    print("Error:", e)
