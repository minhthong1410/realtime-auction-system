# Real-time Auction System

A full-stack real-time auction platform built with Go and Next.js. Features live bidding via WebSocket, TOTP two-factor authentication, Stripe payments, and S3 image uploads.

## Tech Stack

**Backend:** Go 1.24 / Gin / MySQL 8 / Redis / WebSocket (gorilla) / sqlc / Stripe SDK / AWS S3

**Frontend:** Next.js 16 / React 19 / TypeScript / Tailwind CSS 4 / shadcn/ui / Zustand / Axios

**Infrastructure:** Docker Compose / GitHub Actions CI / Railway (backend) / Vercel (frontend)

## Architecture

```
┌─────────────┐     WebSocket      ┌──────────────────┐     Redis Pub/Sub
│   Next.js   │◄──────────────────►│    Go Backend    │◄──────────────────►  Redis
│  (Vercel)   │     REST API       │   (Railway)      │
└─────────────┘◄──────────────────►│                  │───► MySQL
                                   │                  │───► S3 (Cloudflare R2)
                                   │                  │───► Stripe API
                                   └──────────────────┘
```

## Features

- **Real-time bidding** - WebSocket push for new bids, auction end, and balance updates
- **TOTP 2FA** - Mandatory two-factor authentication with QR code setup and 10 backup codes
- **Stripe deposits** - Checkout session flow with webhook confirmation
- **Image uploads** - S3-compatible storage with MIME validation (JPEG/PNG/WebP, max 5MB)
- **Auction lifecycle** - Create → bid → auto-close via background worker (every 2s)
- **Transaction safety** - Row-level locking (`SELECT FOR UPDATE`) on bids, atomic balance operations
- **Multi-instance ready** - Redis pub/sub for WebSocket broadcast across instances

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
| GET | `/api/auctions` | No | List active auctions (paginated) |
| GET | `/api/auctions/:id` | No | Get auction detail |
| GET | `/api/auctions/:id/bids` | No | Bid history (paginated) |
| POST | `/api/auctions` | Yes | Create auction |
| POST | `/api/auctions/:id/bid` | Yes | Place bid |
| GET | `/api/my/auctions` | Yes | List user's auctions |

### Wallet
| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/api/wallet/deposit` | Yes | Create Stripe checkout session |
| GET | `/api/wallet/deposits` | Yes | List deposit history |

### Other
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/upload/image` | Upload auction image (multipart) |
| POST | `/webhook/stripe` | Stripe webhook callback |
| GET | `/ws` | WebSocket upgrade (token optional) |
| GET | `/health` | Health check (DB + Redis ping) |

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
| `balance_update` | `user:*` | `balance`, `reason` |

Room authorization: `auction:*` rooms are public; `user:*` rooms require matching JWT user ID.

## Database Schema

```sql
users       (id BINARY(16), username, email, password, balance BIGINT,
             totp_secret, totp_enabled, backup_codes JSON)

auctions    (id, seller_id FK, title, description, image_url,
             starting_price, current_price, winner_id FK,
             status SMALLINT [1=active, 2=ended, 3=cancelled],
             start_time, end_time)

bids        (id, auction_id FK, user_id FK, amount, created_at)
             -- INDEX (auction_id, amount DESC) for highest bid lookup

deposits    (id, user_id FK, amount, status [pending/completed/failed],
             stripe_payment_id UNIQUE, created_at, updated_at)
```

All monetary amounts stored in **cents** (BIGINT) to avoid floating-point issues. Primary keys are **BINARY(16) UUIDs**.

## Getting Started

### Prerequisites

- Go 1.24+
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

### Docker Compose (Full Stack)

```bash
docker compose up --build
```

Backend at `http://localhost:8080`, frontend at `http://localhost:3000`.

## Project Structure

```
backend/
├── cmd/api/              # Entry point
├── db/
│   ├── migrations/       # SQL migration files (5 migrations)
│   ├── queries/          # sqlc query definitions
│   └── sqlc.yaml         # sqlc config
├── internal/
│   ├── app/api/          # Application setup & routing
│   ├── config/           # Environment config loader
│   ├── errors/           # Custom error types with HTTP status codes
│   ├── handler/          # HTTP handlers (auth, auction, deposit, upload, webhook, ws)
│   ├── i18n/             # Internationalization (en, vi) - embedded
│   ├── middleware/        # Auth, rate limit, CORS, security headers, logging
│   ├── repository/       # sqlc-generated DB layer
│   ├── service/          # Business logic (auth, bid, auction, deposit, totp)
│   ├── storage/          # S3 upload client
│   ├── util/             # UUID & crypto helpers
│   ├── worker/           # Auction closer background job
│   └── ws/               # WebSocket hub & client (room-based, Redis pub/sub)

frontend/
├── src/
│   ├── app/              # Next.js pages (home, login, register, totp-setup,
│   │                     #   verify-otp, auctions/create, auctions/[id],
│   │                     #   my/auctions, wallet, profile)
│   ├── components/       # Navbar, AuctionCard, shadcn/ui components
│   ├── hooks/            # useWebSocket, useCountdown, useBalanceSync
│   ├── lib/              # API client (Axios + interceptors), types, formatters
│   └── stores/           # Zustand auth store
```

## Security

- **Passwords:** bcrypt (cost 12)
- **JWT:** HMAC-SHA256, access token 15min, refresh token 7 days
- **TOTP secrets:** AES-256-GCM encrypted at rest
- **Backup codes:** bcrypt hashed, single-use
- **Rate limiting:** 100 req/min per IP
- **SQL injection:** parameterized queries via sqlc
- **CORS:** configurable allowed origins
- **Security headers:** applied via middleware
- **Image upload:** MIME validation + filename sanitization + 5MB limit
- **Stripe webhook:** signature verification

## Key Design Decisions

- **Amounts in cents** - All monetary values stored as BIGINT cents to avoid floating-point precision errors
- **Row-level locking for bids** - `SELECT FOR UPDATE` prevents race conditions on concurrent bids
- **Atomic balance operations** - Deduct bidder + refund previous bidder in single transaction
- **Redis pub/sub for WebSocket** - Enables horizontal scaling across multiple backend instances
- **Mandatory TOTP** - Every user must set up 2FA during registration (security-first approach)
- **Embedded i18n** - Locale files compiled into binary via `go:embed`, no filesystem dependency
- **sqlc over ORM** - Type-safe generated code from SQL, no runtime query building overhead

## CI/CD

GitHub Actions runs on push/PR to `main`:
- **Backend:** `go build` + `go vet`
- **Frontend:** `npm run lint` + `npm run build`

Production deploys automatically on push to `main`:
- Backend → Railway (with MySQL + Redis)
- Frontend → Vercel
