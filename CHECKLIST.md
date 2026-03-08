# Trading Platform – Master Checklist

**Last Updated:** March 6, 2026
**Project Owner:** Thomas Rogers
**Status:** This checklist has been largely superseded by `CLAUDE.md` (build blueprint) and `CODEBASE_ASSESSMENT.md` (current state). Kept for historical reference.

Status markers:
- ⬜ Not started
- 🟡 In progress
- ✅ Complete

---

## Current Baseline (Locked In)

### Infrastructure
- ✅ Raspberry Pi 5 (8GB RAM) provisioned and stable
- ✅ Docker installed and working
- ✅ Docker Compose in use
- ✅ PostgreSQL 17 container running (`trading-db`)
- ✅ Redis 8 container running (`trading-realtime-data-shared`)
- ✅ Redpanda (Kafka-compatible) container running (`trading-redpanda`)
- ✅ Redpanda Console accessible (port 8080)
- ✅ TimescaleDB for OHLCV time-series data
- ✅ GitHub Actions CI/CD building ARM64 Docker images

### Stock-Service (Go)
- ✅ Service exists and connects to PostgreSQL
- ✅ Database migrations written and applied (13 migrations)
- ✅ Models and repositories implemented
- ✅ Unit tests added to repositories
- ✅ REST API endpoints working (health, stocks CRUD, feedback, tiers)
- ✅ Kafka consumer for `trading.orders` and `trading.positions` topics
- ✅ Raw trades stored to `raw_trades` table
- ✅ Position aggregation logic (BUY creates/updates, SELL closes)
- ✅ Trade history archival on position close
- ✅ Docker image built for ARM64 and published to GHCR
- ✅ Signal feedback persistence (Telegram → REST API → PostgreSQL)
- ✅ Tier ranking storage (`backtest_tiers` table + Redis cache + REST API)

### Robinhood-Sync (Python)
- ✅ Service created and running on Raspberry Pi
- ✅ Connects to Robinhood API (robin_stocks library)
- ✅ Detects filled orders
- ✅ Publishes `TRADE_DETECTED` events to `trading.orders` topic
- ✅ Redis deduplication (tracks synced order IDs)
- ✅ Market hours scheduling (4am-8pm ET, Mon-Fri)
- ✅ Continuous and single-run modes supported
- ✅ Historical sync capability (`--days N`)
- ✅ Earnings calendar sync → Redis
- ✅ Position and watchlist sync → Kafka

---

## Architectural Rules (Non-Negotiable)

- ✅ Services communicate via **Kafka (Redpanda)**, not direct DB calls
- ✅ PostgreSQL is the **source of truth**, not Redis
- ✅ Redis is for **hot / ephemeral data only** (deduplication, caching, tier cache, market context)
- ✅ Each service has **one primary responsibility**
- ✅ No service reaches into another service's database tables
- ✅ Events are immutable (no edits, only new facts)

---

## Kafka Topics ✅ ALL ACTIVE

- ✅ `stock.quotes` (price updates from Alpaca IEX)
- ✅ `stock.indicators` (calculated indicators)
- ✅ `market.context` (regime, VIX, HY spreads, sector strength)
- ✅ `trading.decisions` (decision-engine signals with trade plan + checklist + tier data)
- ✅ `trading.rankings` (daily symbol rankings)
- ✅ `trading.orders` (trade events from Robinhood)
- ✅ `trading.positions` (position sync from Robinhood)
- ✅ `trading.watchlist` (watchlist sync from Robinhood)

---

## Market Data Ingestion ✅ COMPLETE

- ✅ market-data-ingestion service (Go) — Alpaca free IEX real-time feed
- ✅ Publishes to `stock.quotes`
- ✅ TimescaleDB persistence (OHLCV 1-min and daily)
- ✅ Watchlist sync from Robinhood
- ✅ 181 symbols monitored

## Analytics Service ✅ COMPLETE

- ✅ RSI, MACD, SMA (20/50/200), Bollinger Bands, ATR
- ✅ Stochastic (K/D), ADX, EMA (9/21)
- ✅ NaN/Inf rejection
- ✅ Same calculation code in production and backtesting

## Context Service ✅ COMPLETE

- ✅ Regime detection from SPY/sector ETFs
- ✅ VIX + HY credit spreads via FRED API (free)
- ✅ Sector relative strength scoring
- ✅ Publishes every 15 min during market hours

## Decision Engine ✅ COMPLETE

- ✅ Rule evaluation with consensus gate
- ✅ Confidence scoring with regime multiplier
- ✅ Trade plan generation (entry, stop, targets, R:R, position sizing)
- ✅ Pre-trade checklist (5 gates: stop, size, R:R, earnings, regime)
- ✅ Tier ranking integration (confidence + position size multipliers)
- ✅ F-tier blacklist suppression
- ✅ Regime-conditional hard block (BULL-only stocks blocked in non-bull markets)

## Alert Service ✅ COMPLETE

- ✅ Telegram integration with formatted trade plans
- ✅ Pre-trade checklist display
- ✅ Inline feedback buttons (Traded/Skipped)
- ✅ Tier badge in header (e.g., `BUY Signal: WPM [A-tier]`)
- ✅ Regime-conditional warning display

## Trade Journal ✅ COMPLETE

- ✅ Consumes trading.orders
- ✅ Risk metrics at entry (regime + indicators snapshot)
- ✅ P&L tracking with fees

## Backtesting ✅ COMPLETE

- ✅ Multi-TF hybrid (daily entries + 5-min exits)
- ✅ 4-gate validation (walk-forward, bootstrap, Monte Carlo, regime)
- ✅ 103 symbols validated with tier rankings (S-F)
- ✅ populate_tiers.py bulk uploads to stock-service

---

## Remaining Work

See `CODEBASE_ASSESSMENT.md` "Next Steps" and `QUANT_AUDIT.md` "Tier 2 Recommendations" for current priorities:

- ⬜ Portfolio-level risk limit (max 6-8% total portfolio heat)
- ⬜ Sector concentration limit (max 2 concurrent in same sector)
- ⬜ Drawdown circuit breaker (reduce sizing after 10% drawdown)
- ⬜ Setup Scanner (daily scan, "Next 3 Setups" queue)
- ⬜ Daily performance summaries via Telegram

---

**End of Checklist**
