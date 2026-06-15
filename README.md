# Ledger Backend Go

Backend API untuk sistem e-wallet sederhana. Dibangun dengan Go, Gin, GORM, dan PostgreSQL.

## Fitur

- **Registrasi User** — Buat akun baru + wallet otomatis (atomic)
- **Login + JWT** — Autentikasi berbasis token
- **Top-Up Saldo** — Isi saldo wallet
- **Transfer Uang** — Kirim uang antar user dengan pessimistic locking
- **Riwayat Transaksi** — Lihat semua transaksi masuk & keluar

## Tech Stack

- **Language**: Go 1.25
- **Framework**: Gin v1.12
- **ORM**: GORM v1.31
- **Database**: PostgreSQL 16
- **Auth**: JWT (golang-jwt/jwt/v5)
- **Password**: bcrypt

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## Cara Menjalankan

### 1. Clone repository

```bash
git clone https://github.com/bntngridp/ledger-backend-go.git
cd ledger-backend-go
```

### 2. Jalankan PostgreSQL via Docker

```bash
docker compose up -d
```

Tunggu hingga container healthy:

```bash
docker compose ps
```

### 3. Jalankan aplikasi

```bash
go run ./cmd/api
```

Server berjalan di `http://localhost:8080`.

## Endpoint API

### Public (tanpa token)

| Method | Path | Deskripsi |
|--------|------|-----------|
| POST | `/api/v1/auth/register` | Registrasi user baru |
| POST | `/api/v1/auth/login` | Login, dapat JWT token |

### Protected (perlu header `Authorization: Bearer <token>`)

| Method | Path | Deskripsi |
|--------|------|-----------|
| POST | `/api/v1/topup` | Top-up saldo wallet |
| POST | `/api/v1/transfer` | Transfer uang ke user lain |
| GET | `/api/v1/transactions` | Riwayat transaksi |
| GET | `/ping` | Health check |

## Contoh Penggunaan (cURL)

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"budi","email":"budi@mail.com","password":"secret123"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"budi@mail.com","password":"secret123"}'
```

### Top-Up

```bash
curl -X POST http://localhost:8080/api/v1/topup \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"amount":100000,"notes":"top up pertama"}'
```

### Transfer

```bash
curl -X POST http://localhost:8080/api/v1/transfer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"destination_user_id":"<uuid>","amount":50000,"notes":"bayar makan"}'
```

### Riwayat Transaksi

```bash
curl http://localhost:8080/api/v1/transactions \
  -H "Authorization: Bearer <token>"
```

## Struktur Project

```
ledger-backend-go/
├── cmd/api/main.go              # Entry point + DI wiring + routes
├── internal/
│   ├── domain/                  # Entity, DTO, repository interfaces
│   ├── delivery/                # HTTP handlers (Gin)
│   ├── usecase/                 # Business logic
│   └── repository/              # Data access (GORM)
├── pkg/
│   ├── database/                # DB connection + migration
│   └── middleware/              # JWT auth middleware
├── docker-compose.yaml
├── .env
└── go.mod
```

## Environment Variables

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| DB_HOST | localhost | PostgreSQL host |
| DB_PORT | 5432 | PostgreSQL port |
| DB_USER | postgres | PostgreSQL user |
| DB_PASSWORD | postgres | PostgreSQL password |
| DB_NAME | ledger_db | Nama database |
| DB_SSLMODE | disable | SSL mode |
| JWT_SECRET | (wajib) | Secret key untuk JWT |
| JWT_EXPIRY_HOURS | 24 | Masa berlaku token (jam) |
| PORT | 8080 | Port server |

## Testing

```bash
go test ./...
```
