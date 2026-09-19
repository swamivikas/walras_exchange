import argparse
import json
import time
import sys
import websocket

def on_open(ws):
    print("=================================================================")
    print("🚀 [WALRAS DAEMON] GPU Node connected to Gateway via Reverse Tunnel!")
    print("⚡ Listening for incoming inference & tool execution jobs...")
    print("=================================================================")

def on_message(ws, message):
    """Triggered whenever the Gateway sends a job down this open socket"""
    print(f"\n📥 [JOB RECEIVED FROM GATEWAY]: {message}")
    try:
        data = json.loads(message)
        request_id = data.get("request_id", "req-1")
        prompt = data.get("prompt", "")

        # Simulate GPU inference (or connect to local Ollama/vLLM)
        print(f"⚡ [GPU Computing] Running model on prompt: '{prompt}'...")
        time.sleep(0.3) # 300ms simulated GPU computation

        answer = f"Walras AI Node processed prompt successfully: '{prompt}'. [Result: Verified Clean Execution]"

        # Send response back UP the open WebSocket to the Gateway
        response_payload = {
            "request_id": request_id,
            "response": answer,
            "status": "COMPLETED",
            "prompt_tokens": 12,
            "completion_tokens": 18
        }
        ws.send(json.dumps(response_payload))
        print("📤 [GPU Finished] Result sent back up the tunnel to Gateway!")

    except Exception as e:
        print(f"❌ Error processing job: {e}")

def on_error(ws, error):
    print(f"❌ [Tunnel Error]: {error}")

def on_close(ws, close_status_code, close_msg):
    print(f"🔌 [Tunnel Closed]: Code={close_status_code}, Msg={close_msg}")

def start_daemon(url, api_key):
    print(f"Connecting to Walras Gateway at {url}...")
    headers = [f"Authorization: Bearer {api_key}"]
    
    ws = websocket.WebSocketApp(
        url,
        header=headers,
        on_open=on_open,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close,
    )
    # Reconnect loop with 30s ping heartbeats
    ws.run_forever(ping_interval=30, ping_timeout=10)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Walras Provider Edge Daemon")
    parser.add_argument("--key", required=True, help="Seller Node API Key (walras_live_org_...)")
    parser.add_argument("--url", default="ws://localhost:8080/v1/provider/tunnel", help="Gateway Tunnel URL")
    args = parser.parse_args()

    start_daemon(args.url, args.key)
