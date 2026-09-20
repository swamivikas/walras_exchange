"""
Walras Multi-Node Cluster Launcher
===================================
Spins up multiple GPU / AI Agent worker daemons concurrently over independent
reverse WebSocket tunnels to test multi-tenant fleet routing and pool load balancing.
"""

import argparse
import json
import threading
import time
import sys
import websocket

class NodeWorker(threading.Thread):
    def __init__(self, node_id, api_key, agent_type, model, gateway_url="ws://localhost:8080/v1/provider/tunnel"):
        super().__init__()
        self.node_id = node_id
        self.api_key = api_key
        self.agent_type = agent_type
        self.model = model
        self.gateway_url = gateway_url
        self.ws = None
        self.running = True
        self.daemon = True

    def on_open(self, ws):
        print(f"🟢 [FLEET #{self.node_id}] ONLINE -> Pool: [{self.agent_type}] Model: [{self.model}]")

    def on_message(self, ws, message):
        try:
            data = json.loads(message)
            request_id = data.get("request_id", "req-test")
            prompt = data.get("prompt", "")
            print(f"📥 [FLEET #{self.node_id} | {self.agent_type}] Received Job [{request_id}]: '{prompt[:40]}...'")

            # Simulate computation
            time.sleep(0.15) # 150ms processing latency

            solution = f"[{self.agent_type} // Node #{self.node_id} ({self.model})]: Verified output for '{prompt}'"
            response_payload = {
                "request_id": request_id,
                "response": solution,
                "status": "COMPLETED",
                "prompt_tokens": len(prompt) // 4 + 1,
                "completion_tokens": len(solution) // 4 + 5
            }
            ws.send(json.dumps(response_payload))
            print(f"📤 [FLEET #{self.node_id} | {self.agent_type}] Answer dispatched back up tunnel!")
        except Exception as e:
            print(f"❌ [FLEET #{self.node_id}] Error: {e}")

    def on_error(self, ws, error):
        print(f"⚠️ [FLEET #{self.node_id}] Tunnel Error: {error}")

    def on_close(self, ws, close_status_code, close_msg):
        print(f"🔴 [FLEET #{self.node_id}] Offline")

    def run(self):
        headers = [f"Authorization: Bearer {self.api_key}"]
        self.ws = websocket.WebSocketApp(
            self.gateway_url,
            header=headers,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        self.ws.run_forever(ping_interval=20, ping_timeout=10)

def main():
    parser = argparse.ArgumentParser(description="Walras Multi-Node Fleet Runner")
    parser.add_argument("--keys", nargs="+", help="Space-separated list of Seller Node API Keys")
    parser.add_argument("--url", "--endpoint", dest="url", default="ws://localhost:8080/v1/provider/tunnel", help="Gateway Tunnel URL")
    args = parser.parse_args()

    if not args.keys:
        print("Usage: python cluster_fleet.py --keys <KEY_1> <KEY_2> <KEY_3> ...")
        print("\nExample:")
        print("python cluster_fleet.py --keys walras_live_seller_key1 walras_live_seller_key2 ...")
        sys.exit(1)

    print("=================================================================")
    print(f"🚀 Launching Walras Fleet: {len(args.keys)} GPU Nodes Concurrently")
    print(f"⚡ Connecting to Gateway: {args.url}")
    print("=================================================================")

    threads = []
    for idx, key in enumerate(args.keys, 1):
        worker = NodeWorker(node_id=idx, api_key=key, agent_type=f"NODE_{idx}", model="auto-detect", gateway_url=args.url)
        threads.append(worker)
        worker.start()
        time.sleep(0.1) # Stagger connections slightly

    print(f"\n✅ All {len(args.keys)} worker nodes connected and listening for jobs in parallel!")
    print("Press Ctrl+C to stop all cluster workers.\n")

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nStopping cluster fleet...")

if __name__ == "__main__":
    main()
