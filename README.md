# Ledger Backend (Hybrid Wallet System)

[![Go CI](https://github.com/bntngridp/ledger-backend/actions/workflows/go-ci.yml/badge.svg)](https://github.com/bntngridp/ledger-backend/actions/workflows/go-ci.yml)

Backend API untuk sistem **Hybrid Wallet** multi-asset. Mendukung penyimpanan saldo Fiat (Rupiah/IDR) dan Crypto Stablecoin (USDT, USDC), transfer antar user, swap instan dengan kurs real-time, top-up otomatis via Payment Gateway, dan on-chain deposit/withdrawal di blockchain testnet.

Project ini dibangun dengan arsitektur bersih (**Clean Architecture**) serta menjamin integritas data transaksi keuangan menggunakan **ACID Database Transactions** dan **Pessimistic Locking** (`SELECT ... FOR UPDATE`).

---

## Fitur Utama

### 1. Modul Akun & Multi-Asset Dashboard
- **Registrasi & Login (JWT)** — Membuat akun user baru beserta wallet multi-asset secara atomic.
- **Google OAuth 2.0** — Login cepat menggunakan akun Google.
- **Google Authenticator (2FA / TOTP)** — Autentikasi dua langkah menggunakan Google Authenticator. Melindungi login akun serta tindakan sensitif (seperti transfer & penarikan) via verifikasi kode OTP berbasis waktu (TOTP) yang dienkripsi menggunakan AES-256-GCM pada database.
- **Multi-Asset Dashboard** — Menampilkan ringkasan saldo untuk seluruh aset (`IDR`, `USDT`, `USDC`) beserta estimasi total kekayaan dalam Rupiah.

### 2. Modul Fiat (Rupiah/IDR)
- **Top-Up Otomatis (Midtrans Sandbox)** — Integrasi dengan Snap API (Virtual Account / QRIS).
- **Pembayaran Webhook** — Menerima notifikasi settlement pembayaran secara real-time dengan verifikasi Signature Key SHA-512.
- **Transfer Sesama User** — Transfer saldo Rupiah secara instan dan aman dari race condition.
- **Withdrawal Fiat (Midtrans Iris Sandbox)** — Simulasi penarikan dana ke rekening bank eksternal dengan auto-refund otomatis jika penarikan gagal.

### 3. Modul Crypto (USDT & USDC)
- **Generate Deposit Address** — Membuat alamat deposit EVM unik (Polygon Amoy / Sepolia) untuk setiap user. Kunci privat dienkripsi dengan **AES-256-GCM** sebelum disimpan ke database, dan dibersihkan (`zero-out`) dari memori setelah digunakan.
- **On-Chain Deposit Listener** — Goroutine background yang memantau event transfer ERC-20 secara real-time via **Alchemy Websocket RPC**, lengkap dengan reconnect logic, minimal 3 block confirmations, dan idempotensi `tx_hash`.
- **Withdrawal Crypto** — Mengirimkan stablecoin ke wallet eksternal dengan building, signing, dan broadcasting transaksi EVM secara mandiri.

### 4. Modul Swap / Konversi Aset
- **Binance Public API Rate Feed** — Pengambilan kurs real-time dengan in-memory cache TTL 30 detik untuk menghindari rate limiting.
- **Swap Engine** — Menukar Rupiah <-> Crypto atau Crypto <-> Crypto secara instan dengan pemotongan biaya platform flat 0.5%.

---

## Tech Stack

- **Language**: Go 1.25
- **Framework**: Gin v1.12
- **ORM**: GORM v1.31
- **Database**: PostgreSQL 16
- **Blockchain Interface**: go-ethereum v1.13
- **Payment Gateway**: Midtrans SDK (Core API / Snap / Iris Disbursement)
- **Rate Feed**: Binance Spot API

---

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- Akun Alchemy (untuk endpoint WS RPC blockchain testnet)
- Akun Midtrans Sandbox (Server Key & Iris API Key)

---

## Cara Menjalankan

### 1. Clone Repository & Setup Env
```bash
git clone https://github.com/bntngridp/ledger-backend.git
cd ledger-backend
cp .env.example .env
```
Isi konfigurasi kunci di file `.env` (seperti `JWT_SECRET`, `MIDTRANS_SERVER_KEY`, `MIDTRANS_IRIS_API_KEY`, `ALCHEMY_WS_URL`, dan `CRYPTO_ENCRYPTION_KEY`).

### 2. Jalankan PostgreSQL via Docker Compose
```bash
docker compose up -d
```

### 3. Jalankan Aplikasi
```bash
go run ./cmd/api
```
Server akan berjalan di `http://localhost:8080`.

---

## Endpoint API

### Public (Tanpa Autentikasi)

| Method | Path | Deskripsi |
|--------|------|-----------|
| POST | `/api/v1/auth/register` | Registrasi user baru |
| POST | `/api/v1/auth/login` | Login user (jika 2FA aktif, menghasilkan `pre_auth_token`) |
| POST | `/api/v1/auth/2fa/login` | Menyelesaikan tantangan login 2FA dengan kode OTP |
| GET | `/api/v1/auth/google` | Inisiasi Google OAuth login |
| GET | `/api/v1/auth/google/callback` | Callback Google OAuth |
| POST | `/api/v1/webhooks/midtrans` | Webhook notifikasi pembayaran top-up |
| POST | `/api/v1/webhooks/iris` | Webhook callback status penarikan bank |

### Protected (Menerima header `Authorization: Bearer <token>`)

| Method | Path | Deskripsi |
|--------|------|-----------|
| GET | `/api/v1/wallet/dashboard` | Mengambil info wallet & ringkasan saldo |
| POST | `/api/v1/topup` | Memulai inisiasi top-up fiat (Midtrans Snap) |
| POST | `/api/v1/transfer` | Transfer fiat/crypto sesama pengguna platform * |
| POST | `/api/v1/fiat/withdraw` | Tarik saldo Rupiah ke rekening bank (Iris) * |
| GET | `/api/v1/crypto/address` | Dapatkan/buat deposit address EVM user |
| POST | `/api/v1/crypto/withdraw` | Tarik/kirim crypto ke wallet eksternal * |
| GET | `/api/v1/exchange/rate` | Mendapatkan kurs real-time terkini |
| POST | `/api/v1/exchange/swap` | Konversi/tukar antar saldo aset |
| GET | `/api/v1/transactions` | Riwayat transaksi (dilengkapi pagination & filter) |
| POST | `/api/v1/auth/2fa/enable` | Menghasilkan secret key & QR URL untuk setup 2FA |
| POST | `/api/v1/auth/2fa/verify` | Konfirmasi kode OTP pertama untuk mengaktifkan 2FA |
| POST | `/api/v1/auth/2fa/disable` | Menonaktifkan 2FA dengan validasi kode OTP terkini |

> * **Proteksi Keamanan 2FA**:
> Jika 2FA aktif pada akun Anda, endpoint sensitif bertanda bintang (`*`) wajib menyertakan header tambahan `X-2FA-Code: <6_digit_otp_code>`. Jika tidak dikirimkan, request akan ditolak dengan status HTTP `403 Forbidden`.

---

## API Documentation

### Swagger UI
Swagger dokumentasi dapat diakses secara interaktif melalui:
```
http://localhost:8080/swagger/index.html
```

### Postman Collection
Daftar Postman Collection & Environment JSON lengkap tersedia di dalam direktori `postman/`.

---

## Testing

Untuk menjalankan seluruh unit test usecase:
```bash
go test ./...
```
