# Real-time Auction System

A full-stack real-time auction platform built with Go and Next.js. Features live bidding via WebSocket, TOTP two-factor authentication, Stripe payments, wallet management (deposit/withdrawal), Redis caching, Prometheus metrics with Grafana Cloud, and multi-language support (EN/VI).

## Tech Stack

**Backend:** Go 1.25 / Gin / MySQL 8 / Redis / WebSocket (gorilla) / sqlc / Stripe SDK / Cloudflare R2 / zap logger / Prometheus

**Frontend:** Next.js 16 / React 19 / TypeScript / Tailwind CSS 4 / shadcn/ui (Base-UI) / Zustand / Axios / Lucide icons / Plus Jakarta Sans

**Infrastructure:** Docker Compose / GitHub Actions CI / Railway (backend) / Vercel (frontend) / Grafana Cloud (monitoring)

## Architecture

```
┌─────────────┐     WebSocket      ┌──────────────────┐     Redis Pub/Sub
│   Next.js   │◄──────────────────►│    Go Backend    │◄──────────────────►  Redis (cache + pub/sub)
│  (Vercel)   │     REST API       │   (Railway)      │
└─────────────┘◄──────────────────►│                  │───► MySQL
                                   │                  │───► Cloudflare R2 (images)
                                   │                  │───► Stripe API (payments)
                                   │                  │───► Grafana Cloud (metrics)
                                   └──────────────────┘
                                          │
                                   ┌──────┴──────┐
                                   │  Background │
                                   │   Workers   │
                                   │ (auction    │
                                   │  closer)    │
                                   └─────────────┘
```

## Features

- **Real-time bidding** — WebSocket push for new bids, auction end, and balance updates
- **TOTP 2FA** — Mandatory two-factor auth with QR code setup and 10 backup codes (6-digit numeric)
- **Wallet system** — Stripe deposits + bank withdrawal requests with pending/approval flow
- **Auction lifecycle** — Create → bid → auto-close via background worker → transfer funds to seller
- **Redis caching** — Auction list (15s TTL) and detail (30s TTL) with invalidation on bid/create
- **Prometheus metrics** — HTTP requests, latency, bids, deposits, withdrawals, cache hit rate, Go runtime
- **Grafana Cloud** — Remote write push every 15s, production dashboard with 11 panels
- **Multi-language** — Frontend i18n (EN/VI) with language switcher, API responses match UI locale
- **Image uploads** — Cloudflare R2 (S3-compatible) with MIME validation (JPEG/PNG/WebP, max 5MB)
- **Transaction safety** — Row-level locking (`SELECT FOR UPDATE`) on bids, deposits, withdrawals
- **Multi-instance ready** — Redis pub/sub for WebSocket broadcast across instances

## Security

- **Passwords:** bcrypt (default cost)
- **JWT:** HMAC-SHA256, access token 15min, refresh token 7 days
- **TOTP secrets:** AES-256-GCM encrypted at rest
- **Backup codes:** bcrypt hashed, single-use, 6-digit numeric
- **Hardcoded secrets protection:** `requireEnvOrDev()` — panics in production if JWT_SECRET/TOTP_AES_KEY not set
- **Race condition prevention:**
  - Deposit: `LockDepositByStripeID` (SELECT FOR UPDATE) prevents double-credit from webhook retries
  - Withdrawal: `LockUserForWithdrawal` + pending check inside transaction prevents double withdrawal
  - Bid: `LockAuctionForBid` + `DeductBalance WHERE balance >= amount` atomic check
- **Auth timing attack:** constant-time response via dummy bcrypt hash on user-not-found
- **TOTP replay protection:** Redis `SetNX` with 90s TTL per used code
- **Rate limiting:** 10 req/min on auth endpoints (login/register/verify-otp), 30 req/min on refresh, 100 req/min global
- **Amount limits:** Deposit/withdrawal max $100,000
- **Account number masking:** `****6789` pattern in withdrawal list responses
- **SQL injection:** parameterized queries via sqlc
- **Stripe webhook:** signature verification with `IgnoreAPIVersionMismatch`
- **Security headers:** applied via middleware
- **CORS:** configurable allowed origins

## API Endpoints

### Auth
| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/api/auth/register` | No | Register (username, email, password) |
| POST | `/api/auth/login` | No | Login → temp token for TOTP |
| POST | `/api/auth/refresh` | No | Refresh access token |
| POST | `/api/auth/totp/setup` | No | Generate TOTP QR code (temp token) |
| POST | `/api/auth/totp/confirm` | No | Confirm TOTP → access + refresh tokens |
| POST | `/api/auth/verify-otp` | No | Verify OTP or backup code on login |
| GET | `/api/me` | Yes | Get profile |
| POST | `/api/totp/disable` | Yes | Disable 2FA |

### Auctions
| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/api/auctions` | No | List active auctions (paginated, cached 15s) |
| GET | `/api/auctions/:id` | No | Get auction detail (cached 30s) |
| GET | `/api/auctions/:id/bids` | No | Bid history (paginated) |
| POST | `/api/auctions` | Yes | Create auction |
| POST | `/api/auctions/:id/bid` | Yes | Place bid (self-bid prevented) |
| GET | `/api/my/auctions` | Yes | List user's auctions |

### Wallet
| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/api/wallet/deposit` | Yes | Create Stripe checkout session |
| GET | `/api/wallet/deposits` | Yes | List deposit history |
| POST | `/api/wallet/withdrawal` | Yes | Submit withdrawal request |
| GET | `/api/wallet/withdrawals` | Yes | List withdrawal history |

### Other
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/upload/image` | Upload auction image (multipart) |
| POST | `/webhook/stripe` | Stripe webhook (`checkout.session.completed/expired`) |
| GET | `/ws` | WebSocket upgrade (token optional) |
| GET | `/health` | Health check (DB + Redis ping) |
| GET | `/metrics` | Prometheus metrics endpoint |

## WebSocket Events

Connect to `/ws?token=<jwt>` (token optional for unauthenticated browsing).

**Subscribe to a room:**
```json
{"action": "subscribe", "room": "auction:<uuid>"}
```

**Events received:**

| Event | Room | Payload |
|-------|------|---------|
| `new_bid` | `auction:*` | `auction_id`, `amount`, `username`, `bid_count`, `time_left` |
| `auction_ended` | `auction:*` | `auction_id`, `winner`, `final_price` |
| `balance_update` | `user:*` | `balance`, `reason` (deposit/withdrawal/bid_placed/bid_refund/auction_sold) |

Room authorization: `auction:*` rooms are public; `user:*` rooms require matching JWT user ID.

## Database Schema

```sql
users         (id BINARY(16) PK, username, email, password, balance BIGINT,
               totp_secret, totp_enabled, backup_codes JSON)

auctions      (id PK, seller_id FK, title, description, image_url,
               starting_price, current_price, winner_id FK,
               status SMALLINT [1=active, 2=ended, 3=cancelled],
               start_time, end_time)

bids          (id PK, auction_id FK, user_id FK, amount, created_at)

deposits      (id PK, user_id FK, amount, status, stripe_payment_id, created_at, updated_at)

withdrawals   (id PK, user_id FK, amount, status, bank_name, account_number,
               account_holder, note, reviewed_at, created_at, updated_at)
```

Migrations: `000001_init` → `000002_auctions` → `000003_bids` → `000004_deposits` → `000005_totp` → `000006_withdrawals`

All monetary amounts stored in **cents** (BIGINT). Primary keys are **BINARY(16) UUIDs**.

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js 20+
- MySQL 8
- Redis 7
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### Local Development

**1. Start infrastructure:**
```bash
docker compose up -d mysql redis
```

**2. Run migrations:**
```bash
brew install golang-migrate  # macOS
migrate -path backend/db/migrations \
  -database "mysql://root:root@tcp(localhost:3306)/auction" up
```

**3. Configure backend:**
```bash
cat > backend/.env << 'EOF'
DATABASE_DSN=root:root@tcp(localhost:3306)/auction?parseTime=true&loc=UTC
REDIS_ADDR=localhost:6379
JWT_SECRET=your-secret-key
TOTP_AES_KEY=your-32-byte-aes-key-here!!!!!
TOTP_ISSUER=AuctionSystem
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
S3_ENDPOINT=https://your-account.r2.cloudflarestorage.com
S3_BUCKET=auction
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_PUBLIC_URL=https://pub-xxx.r2.dev
FRONTEND_URL=http://localhost:3000
EOF
```

**4. Start backend:**
```bash
cd backend && go run ./cmd/api/
```

**5. Configure and start frontend:**
```bash
cat > frontend/.env.local << 'EOF'
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
EOF

cd frontend && npm install && npm run dev
```

**6. Test Stripe webhook locally:**
```bash
stripe listen --forward-to localhost:8080/webhook/stripe
# Copy whsec_... to .env STRIPE_WEBHOOK_SECRET
```

### Docker Compose (Full Stack)

```bash
docker compose up --build
```

Backend at `http://localhost:8080`, frontend at `http://localhost:3000`.

## Testing

### Unit Tests
```bash
cd backend && go test ./internal/... -count=1
```

### Integration Tests (requires MySQL + Redis)
```bash
# Create test database
mysql -u root -p -e "CREATE DATABASE auction_test"

# Run integration tests
cd backend && go test -tags=integration ./internal/service/... -v -count=1
```

**Test coverage:**

| Package | Tests | What's Covered |
|---------|-------|----------------|
| `util` | 11 | UUID round-trip, edge cases, AES-256-GCM encrypt/decrypt, tampered ciphertext |
| `cache` | 6 | Redis get/set/del, pattern delete, TTL expiry |
| `config` | 6 | env vars, requireEnvOrDev panic in production, DSN parsing |
| `service` | 28 | TOTP tokens (expired/wrong purpose/none algo), auth timing, random codes, account masking, deposit idempotency, withdrawal race prevention, bid refund logic |
| `middleware` | 14 | Rate limit (allow/block/reset/per-IP), JWT auth (valid/expired/malformed/none algo/missing claims) |
| `model` | 9 | JSON serialization, password hidden, nullable fields |
| `httputil` | 14 | Pagination validation, request context |
| `metrics` | 4 | Init, middleware recording, handler, business metrics |

## Project Structure

```
backend/
├── cmd/api/              # Entry point
├── db/
│   ├── migrations/       # SQL migration files (6 migrations)
│   ├── queries/          # sqlc query definitions
│   └── sqlc.yaml         # sqlc config
├── internal/
│   ├── app/api/          # Application setup & routing
│   ├── cache/            # Redis caching layer (get/set/del/pattern)
│   ├── config/           # Environment config loader (requireEnvOrDev for secrets)
│   ├── errors/           # Custom error types with HTTP status codes
│   ├── handler/          # HTTP handlers (auth, auction, deposit, withdrawal, upload, webhook, ws)
│   ├── httputil/         # Response/request helpers, pagination
│   ├── i18n/             # Internationalization (en, vi) — embedded via go:embed
│   ├── logger/           # Global zap logger package
│   ├── metrics/          # Prometheus metrics + Grafana Cloud remote write
│   ├── middleware/        # Auth JWT, rate limit, CORS, security headers, request logging
│   ├── model/            # Domain models and request/response types
│   ├── repository/       # sqlc-generated DB layer
│   ├── service/          # Business logic (auth, bid, auction, deposit, withdrawal, totp)
│   ├── storage/          # S3/R2 upload client
│   ├── util/             # UUID & AES-GCM crypto helpers
│   ├── worker/           # Auction closer background job (auto-close + transfer to seller)
│   └── ws/               # WebSocket hub & client (room-based, Redis pub/sub)

frontend/
├── src/
│   ├── app/              # Next.js pages
│   ├── components/       # Navbar, AuctionCard, shadcn/ui components
│   ├── hooks/            # useWebSocket, useCountdown, useBalanceSync
│   ├── i18n/             # Client-side i18n (EN/VI dictionaries, useTranslation hook)
│   ├── lib/              # API client (Axios + interceptors), types, formatters
│   └── stores/           # Zustand auth store
```

## Scaling Strategy

This system is designed for horizontal scaling:

- **Redis pub/sub** for WebSocket — broadcast bids/events across multiple backend instances
- **Redis caching** — reduces DB load for hot auction data (15-30s TTL, invalidation on write)
- **Prometheus metrics** — real-time observability (HTTP throughput, latency p95, cache hit rate, business KPIs)
- **Grafana Cloud dashboard** — 11 panels monitoring request rate, status codes, top endpoints, bids, deposits, withdrawals, WebSocket connections, Go runtime
- **DB connection pooling** — 25 max open, 5 idle connections
- **Row-level locking** — `SELECT FOR UPDATE` for concurrent bid/deposit/withdrawal safety
- **Rate limiting** — per-IP, stricter on auth endpoints (10 req/min) vs global (100 req/min)
- **Stateless backend** — JWT-based auth, no server-side sessions
- **Background workers** — auction closer runs independently, can be scaled separately
- **Graceful shutdown** — clean connection draining on SIGTERM

## Key Design Decisions

- **Amounts in cents** — All monetary values stored as BIGINT cents to avoid floating-point precision errors
- **Row-level locking for bids** — `SELECT FOR UPDATE` prevents race conditions on concurrent bids
- **Atomic balance operations** — Deduct bidder + refund previous bidder in single transaction
- **Seller payment on close** — Winning bid amount transferred to seller when auction ends
- **Redis pub/sub for WebSocket** — Enables horizontal scaling across multiple backend instances
- **Mandatory TOTP** — Every user must set up 2FA during registration (security-first approach)
- **Embedded i18n** — Backend locale files compiled into binary via `go:embed`; frontend uses client-side context
- **sqlc over ORM** — Type-safe generated code from SQL, no runtime query building overhead
- **Stripe session ID tracking** — Uses checkout session ID (not PaymentIntent) to handle deferred PI creation
- **Constant-time auth** — Dummy bcrypt hash comparison on user-not-found to prevent timing-based email enumeration

## CI/CD

GitHub Actions runs on push/PR to `main`:
- **Backend:** `go build` + `go vet` + `go test`
- **Frontend:** `npm run lint` + `npm run build`

Production deploys automatically on push to `main`:
- Backend → Railway (with MySQL + Redis)
- Frontend → Vercel
- Metrics → Grafana Cloud (remote write)
