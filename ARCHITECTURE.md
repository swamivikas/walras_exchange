# Walras Exchange: Master System Architecture & Technical Blueprint

===============================================================================
SYSTEM SPECIFICATION: High-Performance Decentralized Compute & MCP Exchange
TARGET RUNTIME:       Go (Gateway Engine) + Python/Go (Worker Daemons)
CORE PROTOCOL:        Model Context Protocol (MCP) / JSON-RPC 2.0
SETTLEMENT LAYER:     In-Memory Atomic Ledger (Micro-USD) + Base L2 / Fiat Rails
DATABASE:             MySQL 8.0 (Asynchronous WAL & Cold Rehydration Record)
===============================================================================

---

## 1. Executive Summary & Core Invariants

Walras Exchange is a low-latency clearinghouse and execution router connecting autonomous AI clients (Buyers) with decentralized compute providers and specialized microservices (Sellers).

### Architectural Invariants
1. **Zero Database Lookups on Live Path**: Active balances, escrow locks, and worker capability routes execute purely in Go RAM (`<50ns` to `<1µs`). MySQL is strictly an asynchronous Write-Ahead Log (WAL) and cold recovery record.
2. **NAT-Traversing Reverse Tunnels**: Local GPUs (RTX 4090, Mac Studio) maintain outbound persistent WebSocket streams (`wss://gateway/v1/provider/tunnel`) to the gateway. The gateway dispatches tasks down these open streams, eliminating port-forwarding requirements.
3. **Polymorphic Worker Execution**: The matching engine treats persistent tunnel workers (home GPUs) and serverless HTTP workers (AWS Lambda / Modal) through a unified execution interface.
4. **Cryptographic & Verified Settlement**: Agents transact using an internal microUSD integer ledger ($1.00 USD = 1,000,000 microUSD). Top-ups accept gasless on-chain USDC (Base L2 via EIP-3009) or enterprise fiat (Stripe / Skyfire).

---

## 2. High-Level Architecture Diagram

```
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │                              BUYER INGRESS LAYER                                       │
 │  • Google Antigravity / Cursor / Claude Desktop (MCP: `tools/list`, `tools/call`)      │
 │  • Autonomous Multi-Agent Frameworks (LangGraph, CrewAI, AutoGen)                      │
 │  • OpenAI-Compatible Chat Ingress (`POST /v1/chat/completions`)                         │
 └───────────────────────────────────────────┬────────────────────────────────────────────┘
                                             │ HTTP / JSON-RPC 2.0
                                             ▼
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │                           WALRAS CENTRAL GATEWAY (Go Engine)                           │
 │                                                                                        │
 │  ┌─────────────────────────────────┐     ┌──────────────────────────────────────────┐  │
 │  │        IN-MEMORY VAULT          │     │        ORDER BOOK & MATCHER              │  │
 │  │  • Atomic CAS Escrow Locks      │     │  • Capability Inverted Index             │  │
 │  │  • Sub-50ns balance mutation    │     │  • Multi-factor ranking:                 │  │
 │  │  • LRU/TTL Cache (15m eviction) │     │    Score = Rep - Price - Latency         │  │
 │  └────────────────┬────────────────┘     └────────────────────┬─────────────────────┘  │
 │                   │                                           │                        │
 │                   └─────────────────────┬─────────────────────┘                        │
 │                                         ▼                                              │
 │                           [ WorkerExecutor Interface ]                                 │
 └─────────────────────────┬───────────────────────────────┬──────────────────────────────┘
                           │                               │
            (Stream Payload down Socket)        (HTTP Keep-Alive POST)
                           ▼                               ▼
 ┌────────────────────────────────────────┐  ┌────────────────────────────────────────┐
 │       TUNNEL WORKER (Model A)          │  │       SERVERLESS WORKER (Model B)      │
 │ • RTX 4090 / Mac Studio / Edge Rig     │  │ • AWS Lambda Function URLs / Modal     │
 │ • Persistent WebSocket Reverse Tunnel  │  │ • Ephemeral auto-scaling compute       │
 │ • 0ms connection setup / Bypasses NAT  │  │ • Reused TLS keep-alive connection pool│
 └────────────────────────────────────────┘  └────────────────────────────────────────┘
```

---

## 3. In-Memory Lifecycle & Latency Budget

```
[ EVENT: BUYER TOOL CALL ]
   │
   ├── 1. In-Memory Escrow Lock (< 50 ns)
   │      `Account.AvailableMicroUSD.CompareAndSwap(avail, avail - maxBid)`
   │
   ├── 2. Capability Order Book Match (< 1 µs)
   │      `Registry.MatchBestWorker(capability, maxBid)` -> Selects Worker Node
   │
   ├── 3. Job Dispatch & Transit
   │      ├── Persistent Tunnel: In-memory socket write (< 1 ms overhead)
   │      └── Serverless HTTP: Warm HTTP/2 keep-alive pool (5–15 ms overhead)
   │
   ├── 4. Worker Execution (Variable by Model)
   │      Worker runs inference / scrape / translation and streams result back
   │
   └── 5. Atomic Settlement (< 50 ns)
          `Vault.Settle(buyerCID, sellerCID, actualCost, platformFee)`
```

### Hot RAM vs Cold MySQL State
- **Active Burst**: Go RAM (`sync.Map` + `atomic.Int64`), ~64 bytes/agent, `<50ns` access latency.
- **Idle Agent (15+ min)**: MySQL Database, evicted from RAM, `~1.5ms` hydrated on cold wake.
- **Disaster Recovery**: MySQL WAL Log, rehydrates snapshot on startup.

---

## 4. Multi-Factor Scoring Formula

The Matcher ranks available provider nodes offering a target tool:

$$\text{Score} = \text{Reputation} - (\alpha \times \text{Price}) - (\beta \times \text{Latency})$$

Where:
- $\text{Reputation}$: Float score ($1.0$ to $5.0$).
- $\text{Price}$: MicroUSD cost scaled ($10,000\text{ microUSD} = 1.0\text{ unit}$).
- $\text{Latency}$: Average latency in ms scaled ($100\text{ms} = 1.0\text{ unit}$).
- $\alpha = 0.5$ (price penalty weight).
- $\beta = 0.2$ (latency penalty weight).

---

## 5. MySQL Database Schema

```sql
-- Walras Exchange Database Schema (MySQL 8.0+)
CREATE DATABASE IF NOT EXISTS walras_exchange;
USE walras_exchange;

-- 1. OWNERS TABLE
CREATE TABLE IF NOT EXISTS owners (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    stripe_customer_id VARCHAR(255),
    default_payment_method_id VARCHAR(255),
    owner_wallet_address VARCHAR(42),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- 2. AGENTS TABLE
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(36) PRIMARY KEY,
    owner_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    agent_type VARCHAR(50) NOT NULL,
    agent_role ENUM('BUYER', 'SELLER', 'HYBRID') NOT NULL,
    api_key_hash VARCHAR(64) UNIQUE NOT NULL,
    agent_wallet_address VARCHAR(42),
    api_endpoint VARCHAR(512),
    balance_micro_usd BIGINT DEFAULT 0,
    escrow_locked_micro_usd BIGINT DEFAULT 0,
    status ENUM('ACTIVE', 'SUSPENDED', 'OFFLINE') DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES owners(id) ON DELETE CASCADE,
    INDEX idx_agents_owner_id (owner_id),
    INDEX idx_agents_api_key_hash (api_key_hash)
) ENGINE=InnoDB;

-- 3. CAPABILITIES TABLE
CREATE TABLE IF NOT EXISTS capabilities (
    id VARCHAR(36) PRIMARY KEY,
    agent_id VARCHAR(36) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    description TEXT,
    json_schema JSON NOT NULL,
    cost_per_call_micro_usd BIGINT NOT NULL,
    avg_latency_ms INT DEFAULT 0,
    reputation_score DECIMAL(3, 2) DEFAULT 5.00,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    INDEX idx_capabilities_tool_name (tool_name),
    INDEX idx_capabilities_agent_id (agent_id)
) ENGINE=InnoDB;

-- 4. LEDGER TRANSACTIONS TABLE
CREATE TABLE IF NOT EXISTS ledger_transactions (
    id VARCHAR(36) PRIMARY KEY,
    buyer_agent_id VARCHAR(36),
    seller_agent_id VARCHAR(36),
    amount_micro_usd BIGINT NOT NULL,
    platform_fee_micro_usd BIGINT DEFAULT 0,
    type VARCHAR(50) NOT NULL,
    reference_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'COMPLETED',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ledger_buyer (buyer_agent_id),
    INDEX idx_ledger_seller (seller_agent_id),
    INDEX idx_ledger_ref (reference_id)
) ENGINE=InnoDB;
```

---

## 6. Recommended Directory Layout

```
walras-exchange/
├── ARCHITECTURE.md             # System Specification & Architecture Blueprint
├── cmd/
│   ├── gateway/                # Main Go Exchange Gateway binary
│   │   └── main.go
│   └── daemon/                 # Provider Edge Daemon (Go version)
│       └── main.go
├── internal/
│   ├── vault/                  # In-memory ledger, CAS escrow, TTL cache, WAL
│   │   ├── account.go
│   │   ├── cache.go
│   │   └── wal.go
│   ├── router/                 # Capability registry, scoring engine, matcher
│   │   ├── registry.go
│   │   └── matcher.go
│   ├── workers/                # Polymorphic executor drivers
│   │   ├── executor.go
│   │   ├── tunnel_worker.go
│   │   └── serverless_worker.go
│   ├── tunnel/                 # WebSocket session pool & multiplexer
│   │   └── pool.go
│   ├── auth/                   # EIP-191 / EIP-3009 / ERC-1271 wallet verifiers
│   │   ├── signature.go
│   │   └── erc1271.go
│   ├── db/                     # MySQL storage integration & cold hydration
│   │   └── mysql.go
│   └── gateway/                # MCP JSON-RPC 2.0 & OpenAI HTTP handlers
│       ├── mcp_handlers.go
│       ├── openai_handlers.go
│       ├── register_handlers.go
│       └── topup_handlers.go
├── pkg/
│   └── protocol/               # Shared JSON schemas, manifests, error codes
│       ├── jsonrpc.go
│       └── mcp.go
├── clients/
│   └── python-daemon/          # Lightweight Python seller client (Ollama/vLLM)
│       ├── daemon.py
│       └── requirements.txt
├── docker-compose.yml          # MySQL 8.0 container setup
├── go.mod
└── go.sum
```
