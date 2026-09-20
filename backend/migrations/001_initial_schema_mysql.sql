-- ===============================================================================
-- Walras Exchange: Master Database Schema (MySQL 8.0+)
-- Description: Two-Legged Marketplace Ledger, Multi-Tenant Auth & Type-Based Approval Engine
-- ===============================================================================

CREATE DATABASE IF NOT EXISTS walras_exchange;
USE walras_exchange;

-- -------------------------------------------------------------------------------
-- 1. OWNERS TABLE (Enterprise Clients & Provider Hardware Owners)
-- -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS owners (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    user_type ENUM('ENTERPRISE_BUYER', 'SELLER_OWNER', 'HYBRID') NOT NULL,
    
    -- Enterprise Billing Fields (Post-Paid / Corporate Card)
    stripe_customer_id VARCHAR(255),
    default_payment_method_id VARCHAR(255),
    credit_limit_micro_usd BIGINT DEFAULT 50000000,     -- Default $50.00 free B2B trial credit
    plan_type ENUM('FREE_TRIAL', 'ENTERPRISE_POSTPAID') DEFAULT 'FREE_TRIAL',
    trial_expires_at TIMESTAMP NULL,

    -- Seller Payout Fields (End of Month Disbursements)
    stripe_connect_account_id VARCHAR(255),
    payout_preference ENUM('STRIPE_CONNECT', 'CRYPTO_USDC', 'BANK_WIRE') DEFAULT 'STRIPE_CONNECT',
    owner_wallet_address VARCHAR(42),                   -- EVM address for USDC on Base L2

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_owners_email (email),
    INDEX idx_owners_wallet (owner_wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -------------------------------------------------------------------------------
-- 2. AGENTS TABLE (Buyer Agents, Edge GPUs, Reverse Tunnels)
-- -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(36) PRIMARY KEY,
    owner_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    
    -- Standardized Dropdown Type for Target Benchmark Engine
    agent_type ENUM(
        'MATH_SOLVER',
        'OCR_DOCUMENT_PARSER',
        'CODE_GENERATOR',
        'NLP_TRANSLATOR',
        'GENERAL_LLM_INFERENCE',
        'WEB_SCRAPER',
        'CUSTOM_MICROSERVICE'
    ) NOT NULL DEFAULT 'GENERAL_LLM_INFERENCE',
    
    max_concurrency INT NOT NULL DEFAULT 1,
    vram_gb INT DEFAULT 0,
    agent_role ENUM('BUYER', 'SELLER', 'HYBRID') NOT NULL,
    api_key_hash VARCHAR(64) UNIQUE NOT NULL,           -- SHA-256 hash of API key
    api_key_prefix VARCHAR(64) DEFAULT NULL,            -- e.g. 'walras_live_org_...' for UI display
    api_endpoint VARCHAR(512) DEFAULT NULL,             -- Optional for serverless HTTP workers
    
    -- In-Memory Sync & Approval Fields
    balance_micro_usd BIGINT DEFAULT 0,                 -- Accrued earnings or balance
    escrow_locked_micro_usd BIGINT DEFAULT 0,
    status ENUM('ACTIVE', 'SUSPENDED', 'OFFLINE') DEFAULT 'ACTIVE',
    approval_status ENUM('PENDING_REVIEW', 'APPROVED', 'REJECTED') DEFAULT 'PENDING_REVIEW',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES owners(id) ON DELETE CASCADE,
    INDEX idx_agents_owner_id (owner_id),
    INDEX idx_agents_api_key_hash (api_key_hash),
    INDEX idx_agents_type (agent_type),
    INDEX idx_agents_status (status),
    INDEX idx_agents_approval (approval_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -------------------------------------------------------------------------------
-- 3. CAPABILITIES TABLE (Tools & Models registered by Sellers)
-- -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS capabilities (
    id VARCHAR(36) PRIMARY KEY,
    agent_id VARCHAR(36) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,                    -- e.g. 'deepseek-r1', 'web-scraper'
    description TEXT,
    json_schema JSON NOT NULL,                          -- Parameter schema (MCP standard)
    input_token_price_micro_usd BIGINT NOT NULL DEFAULT 0,
    output_token_price_micro_usd BIGINT NOT NULL DEFAULT 0,
    fixed_cost_per_call_micro_usd BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms INT DEFAULT 0,
    reputation_score DECIMAL(3, 2) DEFAULT 5.00,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    INDEX idx_capabilities_tool_name (tool_name),
    INDEX idx_capabilities_agent_id (agent_id),
    INDEX idx_capabilities_reputation (reputation_score),
    INDEX idx_capabilities_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -------------------------------------------------------------------------------
-- 4. LEG 1: BUYER USAGE TRANSACTIONS (Enterprise -> Compute Consumption)
-- Records compute usage + enterprise surcharge fee. Used for monthly enterprise invoices.
-- -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS buyer_usage_transactions (
    id VARCHAR(36) PRIMARY KEY,
    enterprise_owner_id VARCHAR(36) NOT NULL,
    buyer_agent_id VARCHAR(36) NOT NULL,
    seller_agent_id VARCHAR(36) NOT NULL,
    capability_id VARCHAR(36) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    
    -- Dual-Fee Breakdown for Enterprise
    base_compute_micro_usd BIGINT NOT NULL,             -- Raw compute cost (e.g. $1.00 = 1,000,000 microUSD)
    enterprise_fee_micro_usd BIGINT NOT NULL,           -- Enterprise platform fee (e.g. 3% = 30,000 microUSD)
    total_charged_micro_usd BIGINT NOT NULL,            -- Total billed to enterprise (e.g. $1.03 = 1,030,000 microUSD)
    
    latency_ms INT DEFAULT 0,
    prompt_tokens INT DEFAULT 0,
    completion_tokens INT DEFAULT 0,
    billing_cycle_id VARCHAR(20) NOT NULL,             -- e.g. '2026-09' (for monthly enterprise invoice)
    status ENUM('SETTLED', 'FAILED', 'REFUNDED') DEFAULT 'SETTLED',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (enterprise_owner_id) REFERENCES owners(id) ON DELETE CASCADE,
    FOREIGN KEY (buyer_agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    FOREIGN KEY (capability_id) REFERENCES capabilities(id) ON DELETE CASCADE,
    INDEX idx_buyer_tx_owner (enterprise_owner_id),
    INDEX idx_buyer_tx_cycle (billing_cycle_id),
    INDEX idx_buyer_tx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -------------------------------------------------------------------------------
-- 5. LEG 2: SELLER PAYOUT TRANSACTIONS (Walras Company -> Seller Hardware Owners)
-- Records month-end disbursements with 1% seller take-fee deducted.
-- -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS seller_payout_transactions (
    id VARCHAR(36) PRIMARY KEY,
    seller_owner_id VARCHAR(36) NOT NULL,
    
    -- Dual-Fee Breakdown for Seller
    gross_compute_micro_usd BIGINT NOT NULL,            -- Gross compute earnings generated by seller
    seller_fee_micro_usd BIGINT NOT NULL,               -- Seller take-fee deducted by Walras (e.g. 1%)
    net_payout_micro_usd BIGINT NOT NULL,               -- Net disbursed to seller (gross - seller_fee)
    
    payout_method ENUM('STRIPE_CONNECT', 'CRYPTO_USDC', 'BANK_WIRE') NOT NULL,
    external_reference_id VARCHAR(255),                -- Stripe Transfer ID or Base L2 Tx Hash
    billing_cycle_id VARCHAR(20) NOT NULL,             -- e.g. '2026-09'
    payout_status ENUM('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED') DEFAULT 'PENDING',
    disbursed_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (seller_owner_id) REFERENCES owners(id) ON DELETE CASCADE,
    INDEX idx_seller_payout_owner (seller_owner_id),
    INDEX idx_seller_payout_cycle (billing_cycle_id),
    INDEX idx_seller_payout_status (payout_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
