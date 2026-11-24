# RoomFlow Backend

Go + PostgreSQL implementation of the 33 REST endpoints in
[`doc/后端 API 接口设计文档.md`](../doc/后端%20API%20接口设计文档.md).
Customers, hotel managers and the AI Agent all call the same `/api/v1` API.

## Requirements

- Go 1.25+
- PostgreSQL 15+ (the schema uses composite foreign keys; tested on PostgreSQL 16)

## Run

Create the database once (adjust the role to match `DATABASE_URL`):

```bash
createuser roomflow --pwprompt
createdb -O roomflow roomflow
```

Then start the API:

```bash
cd backend
cp .env.example .env          # then set DATABASE_URL and JWT_SECRET
go run ./cmd/server           # applies pending migrations automatically
```

The server listens on `PORT` (default `8080`) and exposes `GET /health`.

Seed demo data (idempotent):

```bash
go run ./cmd/seed
```

| Role | Email | Password |
|---|---|---|
| manager | `manager@roomflow.local` | `Manager123!` |
| customer | `customer@roomflow.local` | `Customer123!` |

The seed hotel is **RoomFlow 西湖酒店** in 杭州 with 9 rooms across three room types.

## Layout

```
cmd/server/            HTTP server entrypoint
cmd/seed/              demo data seeding
internal/auth/         JWT issuing/parsing, bcrypt password helpers
internal/user/         registration, login, profile, password
internal/hotel/        hotel CRUD, manager authorization
internal/room/         room types and physical rooms
internal/availability/ availability search and price quotes
internal/booking/      bookings, cancellation, check-in/out, inventory locking
internal/statistics/   dashboard and occupancy statistics
internal/middleware/   CORS, logging, recovery, JWT auth, role guards
internal/httpx/        response envelope, error model, request helpers
internal/database/     pool + migration runner
migrations/            embedded SQL migrations
api/openapi.yaml       OpenAPI 3.1 contract
```

## API conventions

- All business routes are under `/api/v1`.
- Authentication: `Authorization: Bearer <JWT>`.
- Every response is `{"code":"OK","message":"success","data":...}`;
  failures use a machine code such as `ROOM_NOT_AVAILABLE`, a Chinese
  message and `data: null`.
- Money is a decimal string (`"328.00"`), dates are `YYYY-MM-DD`, system
  timestamps are RFC 3339 with timezone.
- Hotel/room-type/room statuses:
  - hotel: `active` (listed) / `inactive` (delisted)
  - room type: `active` / `inactive`
  - room: `active` / `maintenance` / `inactive`
- Bookings are made against a room type; a physical room is assigned at
  check-in. Inventory is the count of rooms with status `active` minus
  bookings in `confirmed` or `checked_in` whose date range overlaps
  (`existing.check_in < requested.check_out AND existing.check_out > requested.check_in`).
- Booking creation locks the room-type row with `SELECT ... FOR UPDATE`
  before checking inventory, so concurrent requests cannot oversell.
- No additional fees are implemented; quote `fees` is always `0.00`.

## Verification

```bash
go build ./...
go vet ./...
```

For a live smoke test, start PostgreSQL, run the server and seed, then use
`api/openapi.yaml` as the contract reference.
