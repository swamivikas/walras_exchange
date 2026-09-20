import threading
import time
import sys
import os

# Add local directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
if hasattr(sys.stderr, 'reconfigure'):
    sys.stderr.reconfigure(encoding='utf-8', errors='replace')

from daemon import start_daemon

GATEWAY_URL = "ws://localhost:8080/v1/provider/tunnel"

SELLER_KEYS = [
    # MATH_SOLVER Nodes
    ("walras_live_org_29c249b6_19a7e53be93ba6b3ddd7e3d43d634159", "MATH_SOLVER (TESTING-20-SEPT-1)"),
    ("walras_live_org_29c249b6_e9af3aebd659bd5f3ab4df4bdb3fbcd4", "MATH_SOLVER (TESTING-20-SEPT-2)"),
    ("walras_live_org_29c249b6_ba46564c0cc09c796ac1dc0f9ee1c234", "MATH_SOLVER (TESTING-20-SEPT-3)"),
    ("walras_live_org_29c249b6_10bec550f4bdcb3d896354f6ceebd96a", "MATH_SOLVER (TESTING-20-SEPT-4)"),

    # CODE_GENERATOR Nodes
    ("walras_live_org_5b7a6727_e846263b4e66985dbcd3edaab3bd4565", "CODE_GENERATOR (TESTING-20-SEPT-Vikas_1)"),
    ("walras_live_org_5b7a6727_5267c2b4d96ba6de6b4fb2376221e490", "CODE_GENERATOR (TESTING-20-SEPT-Vikas_2)"),
    ("walras_live_org_5b7a6727_30dae07e47dcce7e10f6298bd020d181", "CODE_GENERATOR (TESTING-20-SEPT-Vikas_4)"),

    # OCR_DOCUMENT_PARSER Nodes (All 10 nodes active)
    ("walras_live_org_5b7a6727_9393f9e4cf22915b282cba6c45e9471c", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_6)"),
    ("walras_live_org_5b7a6727_0a478b3ea16b5a5f0d894cebb4f9f317", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_10)"),
    ("walras_live_org_5b7a6727_88fc2257771fdfceebed7e5587e64ef0", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_8)"),
    ("walras_live_org_5b7a6727_03de42160412bd62c6a3819a54b1a639", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_7)"),
    ("walras_live_org_5b7a6727_e68e7881b8d61c3c914d77971bad0396", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_5)"),
    ("walras_live_org_5b7a6727_bcc3168d31050cabbc88fb4a0e777ce4", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-Vikas_4)"),
    ("walras_live_org_29c249b6_7ae49d6d202e4e4a14ccfae734521962", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-8)"),
    ("walras_live_org_29c249b6_49439f6c2ba986789f91fcf2d9014ee0", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-7)"),
    ("walras_live_org_29c249b6_77ae2a2fc70fcd79625d9c883612e4ef", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-6)"),
    ("walras_live_org_29c249b6_5d91e280243654f3d59263416711684f", "OCR_DOCUMENT_PARSER (TESTING-20-SEPT-5)"),
]

def main():
    print("=================================================================")
    print(f"🚀 [WALRAS FLEET LAUNCHER] Booting {len(SELLER_KEYS)} GPU Nodes concurrently...")
    print(f"⚡ Connecting all reverse tunnels to Gateway: {GATEWAY_URL}")
    print("=================================================================\n")

    threads = []
    for idx, (key, label) in enumerate(SELLER_KEYS, 1):
        print(f"[{idx}/{len(SELLER_KEYS)}] 🔌 Connecting {label}...")
        t = threading.Thread(target=start_daemon, args=(GATEWAY_URL, key), daemon=True)
        t.start()
        threads.append(t)
        time.sleep(0.15) # stagger handshakes slightly

    print("\n✅ All 17 GPU Worker Nodes are LIVE and listening for clearinghouse jobs!")
    print("Press Ctrl+C to stop all nodes.\n")

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n🛑 Shutting down all fleet worker daemons...")

if __name__ == "__main__":
    main()
