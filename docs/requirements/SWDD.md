# Software Design Description (SWDD)
# Hybrid Wallet System — Ledger Backend Go

**Versi Dokumen**: 1.0.0  
**Tanggal**: 2026-07-14  
**Status**: Draft  
**Penulis**: Bintang Ridwan Pribadi  
**Project**: `ledger-backend`  
**Repository**: `github.com/bntngridp/ledger-backend`

---

## Daftar Isi

1. [Pendahuluan](#1-pendahuluan)
2. [Arsitektur Sistem](#2-arsitektur-sistem)
3. [Desain Database](#3-desain-database)
4. [Desain Package & Struktur Folder](#4-desain-package--struktur-folder)
5. [Desain API Endpoint](#5-desain-api-endpoint)
6. [Desain Detail Per Modul](#6-desain-detail-per-modul)
7. [Desain Integrasi Eksternal](#7-desain-integrasi-eksternal)
8. [Penanganan Error](#8-penanganan-error)
9. [Konfigurasi & Environment](#9-konfigurasi--environment)
10. [Keamanan Sistem (Security Design)](#10-keamanan-sistem-security-design)

---

## 1. Pendahuluan

### 1.1 Tujuan Dokumen

Dokumen SWDD (Software Design Description) ini menjabarkan **bagaimana** sistem Hybrid Wallet dibangun secara teknis. Dokumen ini menjadi panduan implementasi bagi developer dan acuan untuk code review, menjamin setiap keputusan desain konsisten dengan persyaratan yang didefinisikan di [SRS.md](./SRS.md).

### 1.2 Referensi

- [SRS.md](./SRS.md) — Software Requirements Specification
- Codebase: `github.com/bntngridp/ledger-backend`

---

## 2. Arsitektur Sistem

### 2.1 Pola Arsitektur: Clean Architecture

Sistem menggunakan pola **Clean Architecture** dengan aturan dependency yang ketat. Dependency hanya boleh mengalir ke dalam (ke domain) dan tidak boleh terbalik:

```
┌──────────────────────────────────────────────────────┐
│                   DELIVERY LAYER                     │
│         (HTTP Handlers / Gin Controllers)            │
│               internal/delivery/                     │
└────────────────────────┬─────────────────────────────┘
                         │ Calls (via interface)
┌────────────────────────▼─────────────────────────────┐
│                   USECASE LAYER                      │
│          (Business Logic / Orchestration)            │
│               internal/usecase/                      │
└────────────────────────┬─────────────────────────────┘
                         │ Calls (via interface)
┌────────────────────────▼─────────────────────────────┐
│                 REPOSITORY LAYER                     │
│        (Data Access: DB, External APIs)              │
│               internal/repository/                  │
└────────────────────────┬─────────────────────────────┘
                         │ Operates on
┌────────────────────────▼─────────────────────────────┐
│                   DOMAIN LAYER                       │
│     (Entities, Interfaces, DTOs — Zero Imports)      │
│               internal/domain/                      │
└──────────────────────────────────────────────────────┘
```

**Aturan mutlak yang tidak boleh dilanggar:**
- **Domain** tidak boleh mengimpor layer mana pun.
- **Repository** tidak boleh mengimpor usecase atau delivery.
- **Usecase** tidak boleh mengimpor delivery atau package `net/http`.
- **Handler/Delivery** tidak boleh berisi logika bisnis.

### 2.2 Komponen Utama Sistem

```
                         ┌─────────────────┐
                         │   Client/User   │
                         └────────┬────────┘
                                  │ HTTPS
                         ┌────────▼────────┐
                         │   Gin Router    │
                         │   + Middleware  │
                         └────────┬────────┘
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
       ┌─────────────┐   ┌────────────────┐   ┌─────────────────┐
       │ Auth Handler│   │  Fiat Handler  │   │ Crypto Handler  │
       └──────┬──────┘   └───────┬────────┘   └────────┬────────┘
              │                  │                      │
              ▼                  ▼                      ▼
       ┌─────────────┐   ┌────────────────┐   ┌─────────────────┐
       │ Auth Usecase│   │  Fiat Usecase  │   │ Crypto Usecase  │
       └──────┬──────┘   └───────┬────────┘   └────────┬────────┘
              │                  │                      │
              └──────────────────┼──────────────────────┘
                                 │
                       ┌─────────▼──────────┐
                       │    Repositories    │
                       │ (User, Wallet,     │
                       │  Transaction,      │
                       │  CryptoAddress)    │
                       └─────────┬──────────┘
                                 │
                       ┌─────────▼──────────┐
                       │    PostgreSQL DB    │
                       └────────────────────┘

Background Services:
  ┌──────────────────────────────────┐
  │   On-Chain Listener (Goroutine)  │──── Alchemy WebSocket RPC
  └──────────────────────────────────┘

  ┌──────────────────────────────────┐
  │   Price Cache Worker (Goroutine) │──── Binance Public API
  └──────────────────────────────────┘
```

---

## 3. Desain Database

### 3.1 Entity Relationship Diagram (ERD)

```
┌─────────────────────────────────────────────────────────────────────┐
│                            users                                    │
│  user_id        UUID      PK                                        │
│  username       VARCHAR(50)  UNIQUE NOT NULL                        │
│  email          VARCHAR(100) UNIQUE NOT NULL                        │
│  password       VARCHAR(255) NULL                                   │
│  google_id      VARCHAR(255) UNIQUE NULL                            │
│  avatar_url     TEXT         NULL                                   │
│  is_active      BOOLEAN    DEFAULT true                             │
│  created_at     TIMESTAMP  DEFAULT NOW()                            │
│  updated_at     TIMESTAMP  DEFAULT NOW()                            │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ 1
                                │
                                │ 1
┌───────────────────────────────▼─────────────────────────────────────┐
│                            wallets                                  │
│  wallet_id      UUID      PK                                        │
│  user_id        UUID      FK(users) UNIQUE NOT NULL                 │
│  created_at     TIMESTAMP  DEFAULT NOW()                            │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ 1
               ┌────────────────┼─────────────────┐
               │ N              │ N               │ N
               ▼                ▼                 ▼
┌──────────────────────┐  ┌──────────────────┐   ┌────────────────────┐
│    wallet_balances   │  │   transactions   │   │  crypto_addresses  │
│  balance_id  UUID PK │  │ transaction_id   │   │  address_id UUID PK│
│  wallet_id   UUID FK │  │   UUID PK        │   │  wallet_id  UUID FK│
│  asset_symbol VARCHAR│  │ source_wallet_id │   │  network   VARCHAR │
│  balance  DECIMAL    │  │   UUID FK NULL   │   │  asset     VARCHAR │
│           (36,18)    │  │ dest_wallet_id   │   │  address   VARCHAR │
│  last_updated        │  │   UUID FK NULL   │   │  enc_private_key   │
│   TIMESTAMP          │  │ asset_symbol     │   │   TEXT NOT NULL    │
│  UNIQUE(wallet_id,   │  │   VARCHAR        │   │  created_at        │
│   asset_symbol)      │  │ amount DECIMAL   │   │  UNIQUE(wallet_id, │
│                      │  │   (36,18)        │   │  network, asset)   │
└──────────────────────┘  │ type    VARCHAR  │   └────────────────────┘
                           │ status  VARCHAR  │
                           │ notes   TEXT     │
                           │ tx_hash VARCHAR  │
                           │   UNIQUE NULL    │
                           │ midtrans_order_id│
                           │   VARCHAR UNIQUE │
                           │   NULL           │
                           │ created_at TIMESTAMP│
                           └──────────────────┘
```

### 3.2 Definisi Tabel Lengkap

#### Tabel: `users`
```sql
CREATE TABLE users (
    user_id    UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   VARCHAR(50)  UNIQUE NOT NULL,
    email      VARCHAR(100) UNIQUE NOT NULL,
    password   VARCHAR(255) NULL, -- NULL jika login via Google
    google_id  VARCHAR(255) UNIQUE NULL, -- Google ID untuk OAuth2
    avatar_url TEXT         NULL, -- Avatar/Profile Picture URL dari Google
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

#### Tabel: `wallets`
> Perubahan dari versi lama: Kolom `balance` dihapus dari sini, dipindahkan ke `wallet_balances`.
```sql
CREATE TABLE wallets (
    wallet_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID UNIQUE NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

#### Tabel: `wallet_balances` *(BARU)*
> Tabel ini menggantikan kolom `balance` di tabel `wallets` untuk mendukung multi-asset.
```sql
CREATE TABLE wallet_balances (
    balance_id   UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id    UUID         NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
    asset_symbol VARCHAR(10)  NOT NULL,  -- 'IDR', 'USDT', 'USDC'
    balance      DECIMAL(36, 18) NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (wallet_id, asset_symbol),
    CHECK (balance >= 0)
);
```

#### Tabel: `transactions` *(MODIFIKASI)*
> Kolom `asset_symbol`, `tx_hash`, dan `midtrans_order_id` ditambahkan.
```sql
CREATE TABLE transactions (
    transaction_id     UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_wallet_id   UUID         REFERENCES wallets(wallet_id),
    dest_wallet_id     UUID         REFERENCES wallets(wallet_id),
    asset_symbol       VARCHAR(10)  NOT NULL,  -- 'IDR', 'USDT', 'USDC'
    amount             DECIMAL(36, 18) NOT NULL CHECK (amount > 0),
    type               VARCHAR(30)  NOT NULL,
    -- Nilai type: 'topup_fiat', 'transfer_fiat', 'withdraw_fiat',
    --             'crypto_deposit', 'crypto_withdrawal', 'swap'
    status             VARCHAR(20)  NOT NULL DEFAULT 'pending',
    -- Nilai status: 'pending', 'success', 'failed'
    transaction_notes  TEXT,
    tx_hash            VARCHAR(100) UNIQUE,  -- Blockchain tx hash (nullable untuk fiat)
    midtrans_order_id  VARCHAR(100) UNIQUE,  -- Midtrans order ID (nullable untuk crypto)
    rate_used          DECIMAL(20, 8),       -- Kurs yang digunakan saat swap (nullable)
    fee_charged        DECIMAL(36, 18),      -- Biaya platform (nullable)
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes untuk performa
CREATE INDEX idx_transactions_source_wallet ON transactions(source_wallet_id);
CREATE INDEX idx_transactions_dest_wallet   ON transactions(dest_wallet_id);
CREATE INDEX idx_transactions_asset         ON transactions(asset_symbol);
CREATE INDEX idx_transactions_type          ON transactions(type);
CREATE INDEX idx_transactions_status        ON transactions(status);
CREATE INDEX idx_transactions_created_at    ON transactions(created_at DESC);
```

#### Tabel: `crypto_addresses` *(BARU)*
```sql
CREATE TABLE crypto_addresses (
    address_id      UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id       UUID         NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
    network         VARCHAR(30)  NOT NULL,  -- 'polygon_amoy', 'sepolia'
    asset_symbol    VARCHAR(10)  NOT NULL,  -- 'USDT', 'USDC'
    address         VARCHAR(42)  NOT NULL,  -- EVM address: 0x...
    enc_private_key TEXT         NOT NULL,  -- Private key terenkripsi AES-256-GCM
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (wallet_id, network, asset_symbol)
);
```

### 3.3 Aset yang Didukung

| Asset Symbol | Tipe | Jaringan | Smart Contract (Amoy Testnet) |
|:---|:---|:---|:---|
| `IDR` | Fiat (Rupiah) | Internal Ledger | - |
| `USDT` | Crypto Stablecoin | Polygon Amoy Testnet | TBD (testnet contract) |
| `USDC` | Crypto Stablecoin | Polygon Amoy Testnet | TBD (testnet contract) |

### 3.4 Aturan Tipe Data Keuangan

> [!IMPORTANT]
> Ini adalah aturan yang WAJIB diikuti untuk menghindari kesalahan perhitungan keuangan.

- **IDR (Rupiah)**: Disimpan sebagai `DECIMAL(36, 18)` dalam satuan **Rupiah penuh** (bukan sen). Meski tipe datanya mendukung desimal, semua nilai IDR harus **bilangan bulat** (tidak ada Rp 500,50). Validasi di layer Usecase.
- **Crypto (USDT/USDC)**: Disimpan sebagai `DECIMAL(36, 18)` — mendukung 18 angka desimal sesuai standar ERC-20.
- **Jangan pernah menggunakan `float32` atau `float64` Go** untuk perhitungan finansial. Gunakan library `shopspring/decimal` di Go untuk semua operasi aritmatika keuangan.

---

## 4. Desain Package & Struktur Folder

### 4.1 Struktur Folder Lengkap

```
ledger-backend-go/
│
├── cmd/
│   └── api/
│       └── main.go                    # Entry point: DI wiring, server start
│
├── internal/
│   │
│   ├── domain/                        # LAYER DOMAIN (Zero Import)
│   │   ├── user.go                    # Entity: User
│   │   ├── wallet.go                  # Entity: Wallet
│   │   ├── wallet_balance.go          # Entity: WalletBalance (BARU)
│   │   ├── transaction.go             # Entity: Transaction (MODIFIKASI)
│   │   ├── crypto_address.go          # Entity: CryptoAddress (BARU)
│   │   ├── dto.go                     # Semua Request/Response DTO
│   │   ├── errors.go                  # Domain-specific error types (BARU)
│   │   └── repository.go             # Semua interface Repository
│   │
│   ├── delivery/                      # LAYER DELIVERY (HTTP Handlers)
│   │   ├── auth_handler.go            # Handler: Register, Login
│   │   ├── fiat_handler.go            # Handler: TopUp, Transfer, Withdraw IDR (BARU)
│   │   ├── crypto_handler.go          # Handler: Deposit Address, Withdraw Crypto (BARU)
│   │   ├── exchange_handler.go        # Handler: Get Rate, Swap (BARU)
│   │   ├── wallet_handler.go          # Handler: Dashboard, History (MODIFIKASI)
│   │   └── webhook_handler.go         # Handler: Midtrans Webhook (BARU)
│   │
│   ├── usecase/                       # LAYER USECASE (Business Logic)
│   │   ├── auth_usecase.go            # Usecase: Register, Login (MODIFIKASI)
│   │   ├── fiat_usecase.go            # Usecase: TopUp, Transfer, Withdraw IDR (BARU)
│   │   ├── crypto_usecase.go          # Usecase: Address, Deposit, Withdraw Crypto (BARU)
│   │   ├── exchange_usecase.go        # Usecase: Get Rate, Swap (BARU)
│   │   ├── wallet_usecase.go          # Usecase: Dashboard, History (MODIFIKASI)
│   │   └── webhook_usecase.go         # Usecase: Proses webhook Midtrans (BARU)
│   │
│   └── repository/                    # LAYER REPOSITORY (Data Access)
│       ├── user_repository.go         # Repo: CRUD User
│       ├── wallet_repository.go       # Repo: CRUD Wallet & WalletBalance (MODIFIKASI)
│       ├── transaction_repository.go  # Repo: CRUD Transaction (MODIFIKASI)
│       └── crypto_address_repository.go # Repo: CRUD CryptoAddress (BARU)
│
├── pkg/
│   ├── config/
│   │   └── config.go                  # Load & validasi semua env vars
│   ├── database/
│   │   └── database.go                # Koneksi DB & auto-migration
│   ├── middleware/
│   │   └── auth.go                    # JWT validation middleware
│   ├── response/
│   │   └── response.go                # Helper: success/error response builder
│   ├── crypto/
│   │   ├── wallet_generator.go        # Generate EVM key pair (BARU)
│   │   └── encryptor.go               # AES-256-GCM encrypt/decrypt (BARU)
│   ├── midtrans/
│   │   └── client.go                  # Midtrans API HTTP client wrapper (BARU)
│   ├── blockchain/
│   │   ├── alchemy_client.go          # Alchemy RPC client (BARU)
│   │   └── erc20_listener.go          # On-chain event listener goroutine (BARU)
│   └── price/
│       └── price_cache.go             # Binance rate fetcher + in-memory cache (BARU)
│
├── docs/
│   ├── requirements/
│   │   ├── SRS.md                     # Software Requirements Specification
│   │   └── SWDD.md                    # Software Design Description (file ini)
│   ├── docs.go                        # Swagger docs
│   └── swagger.yaml
│
├── postman/                           # Postman collection & environment
├── docker-compose.yaml
├── .env
├── .env.example
├── go.mod
└── go.sum
```

### 4.2 Aturan Penamaan File

| Tipe | Konvensi Penamaan | Contoh |
|:---|:---|:---|
| Entity | `<nama_entitas>.go` | `wallet_balance.go` |
| DTO | Semua dalam satu file | `dto.go` |
| Handler | `<modul>_handler.go` | `fiat_handler.go` |
| Usecase | `<modul>_usecase.go` | `crypto_usecase.go` |
| Repository | `<modul>_repository.go` | `crypto_address_repository.go` |
| Test | `<nama_file>_test.go` | `fiat_usecase_test.go` |

---

## 5. Desain API Endpoint

### 5.1 Konvensi Umum

- **Base Path**: `/api/v1`
- **Content-Type**: `application/json`
- **Auth Header**: `Authorization: Bearer <jwt_token>` (untuk endpoint protected)
- **Response Envelope**:
```json
// Success
{
  "status": "success",
  "message": "Deskripsi singkat operasi yang berhasil",
  "data": { }
}

// Success with pagination
{
  "status": "success",
  "message": "...",
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}

// Error
{
  "status": "error",
  "message": "Deskripsi error untuk user",
  "errors": [ { "field": "email", "message": "must be a valid email" } ]
}
```

### 5.2 Tabel Endpoint Lengkap

#### 🔓 Public Endpoints (Tanpa Autentikasi)

| Method | Path | Handler | Deskripsi |
|:---|:---|:---|:---|
| `POST` | `/api/v1/auth/register` | `auth_handler.Register` | Registrasi user & wallet baru |
| `POST` | `/api/v1/auth/login` | `auth_handler.Login` | Login, dapat JWT Token |
| `POST` | `/api/v1/webhooks/midtrans` | `webhook_handler.HandleMidtrans` | Notifikasi pembayaran dari Midtrans |
| `GET` | `/ping` | - | Health check |

#### 🔒 Protected Endpoints (Butuh JWT Token)

| Method | Path | Handler | Deskripsi |
|:---|:---|:---|:---|
| `GET` | `/api/v1/wallet/dashboard` | `wallet_handler.GetDashboard` | Saldo semua aset + estimasi total |
| `GET` | `/api/v1/wallet/transactions` | `wallet_handler.GetHistory` | Riwayat transaksi terpadu |
| `POST` | `/api/v1/fiat/topup/init` | `fiat_handler.InitiateTopUp` | Buat invoice QRIS/VA Midtrans |
| `POST` | `/api/v1/fiat/transfer` | `fiat_handler.Transfer` | Transfer Rupiah ke user lain |
| `POST` | `/api/v1/fiat/withdraw` | `fiat_handler.Withdraw` | Tarik saldo IDR ke rekening bank |
| `GET` | `/api/v1/crypto/address` | `crypto_handler.GetDepositAddress` | Lihat deposit address crypto |
| `POST` | `/api/v1/crypto/withdraw` | `crypto_handler.Withdraw` | Kirim crypto ke wallet eksternal |
| `GET` | `/api/v1/exchange/rate` | `exchange_handler.GetRate` | Kurs real-time IDR/USDT/USDC |
| `POST` | `/api/v1/exchange/swap` | `exchange_handler.Swap` | Tukar antar aset |

---

## 6. Desain Detail Per Modul

### 6.1 Modul: Fiat Top-Up (Midtrans Integration)

**Alur Proses Top-Up QRIS:**
```
User                  API Server                   Midtrans Sandbox
 │                        │                              │
 │  POST /fiat/topup/init │                              │
 │  {amount, method:qris} │                              │
 │───────────────────────►│                              │
 │                        │                              │
 │                        │ Buat txn "pending"            │
 │                        │ di database                   │
 │                        │                              │
 │                        │ POST /charge (Midtrans API)  │
 │                        │──────────────────────────────►
 │                        │                              │
 │                        │◄─────────────────────────────
 │                        │ {qr_string, order_id, expiry}│
 │                        │                              │
 │◄───────────────────────│                              │
 │  {qr_url, expiry_time} │                              │
 │                        │                              │
 │  [User scan QRIS &     │                              │
 │   bayar via GoPay/Bank]│                              │
 │                        │                              │
 │                        │◄─────────────────────────────
 │                        │  POST /webhooks/midtrans     │
 │                        │  {order_id, status:settle}   │
 │                        │                              │
 │                        │ Verifikasi signature          │
 │                        │ Update saldo IDR (atomic)     │
 │                        │ Update txn status "success"  │
 │                        │──────────────────────────────►
 │                        │ 200 OK                       │
```

**Implementasi Signature Verification:**
```go
// pkg/midtrans/client.go
func VerifySignature(orderID, statusCode, grossAmount, serverKey string, receivedSignature string) bool {
    raw := orderID + statusCode + grossAmount + serverKey
    hash := sha512.Sum512([]byte(raw))
    expected := hex.EncodeToString(hash[:])
    return expected == receivedSignature
}
```

### 6.2 Modul: Crypto Deposit (On-Chain Listener)

**Alur On-Chain Listener:**
```go
// pkg/blockchain/erc20_listener.go

// Background Goroutine yang berjalan saat server start
func StartERC20Listener(ctx context.Context, alchemyWsURL string, contractAddress common.Address, repo domain.CryptoAddressRepository, txRepo domain.TransactionRepository) {
    // 1. Connect ke Alchemy WebSocket RPC
    // 2. Subscribe ke event Transfer(from, to, value) dari smart contract USDT/USDC
    // 3. Saat event masuk:
    //    a. Cek apakah "to" address ada di database kita (GetByAddress)
    //    b. Cek apakah tx_hash sudah diproses (idempotency check)
    //    c. Tunggu minimum 5 block confirmations
    //    d. Kredit saldo user secara atomic
    //    e. Simpan tx_hash ke database
    // 4. Handle reconnect dengan exponential backoff jika koneksi putus
}
```

**Alur Deposit Crypto (Diagram):**
```
MetaMask User           Blockchain (Amoy)          API Server (Listener)
     │                        │                          │
     │  Send 10 USDT          │                          │
     │  ke address user       │                          │
     │───────────────────────►│                          │
     │                        │                          │
     │                        │ Event: Transfer(         │
     │                        │   from: 0xUser...,       │
     │                        │   to: 0xDepositAddr...,  │
     │                        │   value: 10000000        │
     │                        │ )                        │
     │                        │─────────────────────────►│
     │                        │                          │
     │                        │  [tunggu 5 blok]         │
     │                        │──────────────────────────►
     │                        │                          │
     │                        │  Cek DB: address match? │
     │                        │  Cek DB: tx_hash baru?  │
     │                        │  Kredit saldo USDT      │
     │                        │  Insert txn record      │
```

### 6.3 Modul: Exchange & Swap

**Alur Swap IDR → USDT:**
```
User                    API Server                 Binance API (Cache)
 │                           │                           │
 │  POST /exchange/swap      │                           │
 │  {from:IDR, to:USDT,      │                           │
 │   amount:162000}          │                           │
 │──────────────────────────►│                           │
 │                           │ Cek cache (TTL: 30s)      │
 │                           │   (jika expired)          │
 │                           │──────────────────────────►│
 │                           │◄──────────────────────────│
 │                           │  Rate: 1 USDT = 16200 IDR │
 │                           │                           │
 │                           │ Hitung:                   │
 │                           │  to_amount = 162000/16200 │
 │                           │            = 10 USDT      │
 │                           │  fee = 10 * 0.5% = 0.05  │
 │                           │  net = 10 - 0.05 = 9.95  │
 │                           │                           │
 │                           │ Atomic DB Transaction:    │
 │                           │  Debit IDR: -162000       │
 │                           │  Kredit USDT: +9.95       │
 │                           │  Insert txn type=swap     │
 │◄──────────────────────────│                           │
 │  {from: 162000 IDR,       │                           │
 │   to: 9.95 USDT,          │                           │
 │   fee: 0.05 USDT,         │                           │
 │   rate: 16200}            │                           │
```

### 6.4 Modul: Wallet Dashboard

**Logika Hitung Estimasi Total Aset (dalam IDR):**
```go
// internal/usecase/wallet_usecase.go
func (uc *walletUsecase) GetDashboard(userID uuid.UUID) (*domain.DashboardResponse, error) {
    balances, _ := uc.walletRepo.GetAllBalances(walletID)

    totalIDR := decimal.Zero
    for _, b := range balances {
        switch b.AssetSymbol {
        case "IDR":
            totalIDR = totalIDR.Add(b.Balance)
        case "USDT":
            rate, _ := uc.priceCache.GetRate("USDT_IDR")
            totalIDR = totalIDR.Add(b.Balance.Mul(rate))
        case "USDC":
            rate, _ := uc.priceCache.GetRate("USDC_IDR")
            totalIDR = totalIDR.Add(b.Balance.Mul(rate))
        }
    }
    // ...
}
```

---

## 7. Desain Integrasi Eksternal

### 7.1 Midtrans Sandbox

**Konfigurasi Client:**
```go
// pkg/midtrans/client.go
type MidtransClient struct {
    ServerKey  string
    BaseURL    string // https://api.sandbox.midtrans.com
    HTTPClient *http.Client
}
```

**Metode yang Disediakan:**
| Method | Deskripsi |
|:---|:---|
| `CreateQRISCharge(amount int64, orderID string)` | Membuat transaksi QRIS |
| `CreateVACharge(amount int64, orderID, bankCode string)` | Membuat transaksi Virtual Account |
| `VerifySignature(orderID, statusCode, grossAmount, signature string)` | Validasi webhook signature |

**Midtrans Payment Method yang Didukung:**

| Metode | Tipe Midtrans | Keterangan |
|:---|:---|:---|
| QRIS | `gopay` (channel: `qris`) | Untuk semua dompet digital via QRIS |
| BCA Virtual Account | `bank_transfer` + `bca` | Transfer ke VA BCA |
| BNI Virtual Account | `bank_transfer` + `bni` | Transfer ke VA BNI |
| Mandiri Virtual Account | `echannel` | Transfer ke VA Mandiri |

### 7.2 Alchemy (Blockchain RPC)

**Konfigurasi:**
```go
// pkg/blockchain/alchemy_client.go
type AlchemyClient struct {
    HTTPRPCURL  string  // https://polygon-amoy.g.alchemy.com/v2/{API_KEY}
    WebSocketURL string // wss://polygon-amoy.g.alchemy.com/v2/{API_KEY}
    EthClient   *ethclient.Client
}
```

**Metode yang Disediakan:**
| Method | Deskripsi |
|:---|:---|
| `GetTokenBalance(address, contractAddr)` | Mendapatkan saldo token ERC-20 |
| `SendSignedTransaction(signedTx)` | Broadcast transaksi ke blockchain |
| `GetBlockNumber()` | Mendapatkan nomor blok terkini |
| `WatchERC20Transfers(contractAddr, handler)` | Subscribe event transfer |

### 7.3 Binance Public API

**Konfigurasi:**
```go
// pkg/price/price_cache.go
type PriceCache struct {
    cache     map[string]CachedRate  // pair -> rate
    ttl       time.Duration          // 30 detik
    mu        sync.RWMutex
    binanceURL string // https://api.binance.com/api/v3
}
```

**Cara Kerja:**
1. Ambil harga `USDT/USDT` dari Binance (biasanya mendekati 1.0 USD).
2. Ambil kurs `USD/IDR` dari sumber lain atau hardcode sebagai konfigurasi (karena Binance tidak langsung menyediakan pair IDR).
3. Kalkulasi: `1 USDT = 1 USD × USD_IDR_RATE`.

---

## 8. Penanganan Error

### 8.1 Domain Error Types

```go
// internal/domain/errors.go
var (
    ErrNotFound           = errors.New("resource not found")
    ErrUnauthorized       = errors.New("unauthorized access")
    ErrForbidden          = errors.New("forbidden")
    ErrConflict           = errors.New("resource already exists")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrInvalidInput       = errors.New("invalid input")
    ErrDuplicateTransaction = errors.New("transaction already processed")
    ErrExternalService    = errors.New("external service error")
    ErrInvalidSignature   = errors.New("invalid webhook signature")
    ErrSelfTransfer       = errors.New("cannot transfer to yourself")
)
```

### 8.2 HTTP Status Code Mapping

| Domain Error | HTTP Status |
|:---|:---|
| `ErrNotFound` | `404 Not Found` |
| `ErrUnauthorized` | `401 Unauthorized` |
| `ErrForbidden` | `403 Forbidden` |
| `ErrConflict` | `409 Conflict` |
| `ErrInsufficientBalance` | `422 Unprocessable Entity` |
| `ErrInvalidInput` | `400 Bad Request` |
| `ErrDuplicateTransaction` | `200 OK` (idempotent, jangan error) |
| `ErrExternalService` | `502 Bad Gateway` |
| `ErrInvalidSignature` | `403 Forbidden` |
| `ErrSelfTransfer` | `400 Bad Request` |
| Error tak terduga lainnya | `500 Internal Server Error` |

### 8.3 Error Middleware (Centralized)

```go
// pkg/response/response.go
func HandleError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        c.JSON(http.StatusNotFound, ErrorResponse{...})
    case errors.Is(err, domain.ErrInsufficientBalance):
        c.JSON(http.StatusUnprocessableEntity, ErrorResponse{...})
    // ... dst
    default:
        // Log error internal, tapi jangan expose ke client
        log.Error(err)
        c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
    }
}
```

---

## 9. Konfigurasi & Environment

### 9.1 Variabel Environment Lengkap

```env
# === DATABASE ===
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ledger_db
DB_SSLMODE=disable

# === SERVER ===
PORT=8080

# === JWT ===
JWT_SECRET=your-super-secret-key-minimum-32-chars
JWT_EXPIRY_HOURS=24

# === MIDTRANS ===
MIDTRANS_SERVER_KEY=SB-Mid-server-xxxxxxxxxxxx
MIDTRANS_CLIENT_KEY=SB-Mid-client-xxxxxxxxxxxx
MIDTRANS_ENV=sandbox  # 'sandbox' atau 'production'

# === ALCHEMY (Blockchain RPC) ===
ALCHEMY_API_KEY=your-alchemy-api-key
ALCHEMY_NETWORK=polygon-amoy  # atau 'eth-sepolia'
ALCHEMY_HTTP_URL=https://polygon-amoy.g.alchemy.com/v2/your-api-key
ALCHEMY_WS_URL=wss://polygon-amoy.g.alchemy.com/v2/your-api-key

# === CRYPTO SMART CONTRACTS (Testnet) ===
USDT_CONTRACT_ADDRESS=0x...  # USDT contract di Polygon Amoy
USDC_CONTRACT_ADDRESS=0x...  # USDC contract di Polygon Amoy

# === CRYPTO ENCRYPTION ===
# 32-byte key untuk AES-256-GCM (base64 encoded)
CRYPTO_ENCRYPTION_KEY=base64encodedkeyof32bytes==

# === PRICE FEED ===
BINANCE_API_URL=https://api.binance.com/api/v3
USD_IDR_RATE=16200  # Fallback rate jika Binance tidak tersedia

# === SWAP FEE ===
SWAP_FEE_PERCENTAGE=0.5  # 0.5% biaya platform
```

---

## 10. Keamanan Sistem (Security Design)

### 10.1 Keamanan Private Key Crypto

```
Saat Generate Address:
  1. go-ethereum generate random key pair
  2. Public address disimpan plaintext di tabel crypto_addresses
  3. Private key → AES-256-GCM encrypt (dengan CRYPTO_ENCRYPTION_KEY dari env)
  4. Encrypted private key disimpan ke kolom enc_private_key

Saat Signing Transaksi (Withdraw Crypto):
  1. Ambil enc_private_key dari DB
  2. AES-256-GCM decrypt menggunakan CRYPTO_ENCRYPTION_KEY
  3. Gunakan private key untuk sign transaksi
  4. Segera zero-out variabel private key dari memori
  5. Broadcast signed transaction ke blockchain
  6. Private key tidak pernah keluar dari server
```

### 10.2 Keamanan Webhook Midtrans

```
Request masuk ke POST /api/v1/webhooks/midtrans:
  1. Ambil header signature dari payload
  2. Rekonstruksi string: order_id + status_code + gross_amount + SERVER_KEY
  3. Hitung SHA512 dari string tersebut
  4. Bandingkan dengan signature yang diterima (constant-time comparison)
  5. Jika tidak sama → 403 Forbidden, log percobaan akses tidak sah
  6. Jika sama → proses payload
```

### 10.3 Idempotency Keys

| Operasi | Idempotency Key | Implementasi |
|:---|:---|:---|
| Midtrans Top-Up | `midtrans_order_id` | `UNIQUE` constraint di kolom `transactions.midtrans_order_id` |
| Crypto Deposit | `tx_hash` blockchain | `UNIQUE` constraint di kolom `transactions.tx_hash` |

### 10.4 Input Validation Summary

| Field | Validasi |
|:---|:---|
| `email` | Format email valid (binding: `email`) |
| `password` | Minimum 6 karakter (binding: `min=6`) |
| `amount` (IDR) | Harus > 0, bilangan bulat, minimum sesuai jenis transaksi |
| `amount` (Crypto) | Harus > 0, maksimum 18 desimal |
| `to_address` (Crypto) | Panjang 42 karakter, berawalan `0x`, hexadecimal valid |
| `destination_user_id` | Format UUID valid, tidak sama dengan sender |
| `asset_symbol` | Hanya dari whitelist: `IDR`, `USDT`, `USDC` |
| `network` | Hanya dari whitelist: `polygon_amoy`, `sepolia` |
