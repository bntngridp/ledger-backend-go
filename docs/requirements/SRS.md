# Software Requirements Specification (SRS)
# Hybrid Wallet System — Ledger Backend Go

**Versi Dokumen**: 1.0.0  
**Tanggal**: 2026-07-14  
**Status**: Draft  
**Penulis**: Bintang Ridwan Pribadi  
**Project**: `ledger-backend-go`  
**Repository**: `github.com/bntngridp/ledger-backend-go`

---

## Daftar Isi

1. [Pendahuluan](#1-pendahuluan)
2. [Deskripsi Keseluruhan Sistem](#2-deskripsi-keseluruhan-sistem)
3. [Persyaratan Fungsional](#3-persyaratan-fungsional)
4. [Persyaratan Non-Fungsional](#4-persyaratan-non-fungsional)
5. [Batasan Sistem](#5-batasan-sistem)
6. [Asumsi dan Ketergantungan Eksternal](#6-asumsi-dan-ketergantungan-eksternal)
7. [Glossary](#7-glossary)

---

## 1. Pendahuluan

### 1.1 Tujuan Dokumen

Dokumen SRS ini mendefinisikan seluruh persyaratan fungsional dan non-fungsional untuk sistem **Hybrid Wallet** yang akan dikembangkan di atas project `ledger-backend-go`. Dokumen ini menjadi **kontrak resmi** antara developer dan sistem agar pengembangan tetap konsisten, tidak melenceng dari target, dan bisa digunakan sebagai acuan saat review atau debugging di kemudian hari.

### 1.2 Ruang Lingkup (Scope)

Sistem **Hybrid Wallet** adalah platform backend API berbasis Go yang memungkinkan user untuk:

- Menyimpan dan mengelola saldo dalam mata uang Fiat **(IDR/Rupiah)**.
- Menyimpan dan mengelola saldo dalam aset Crypto Stablecoin **(USDT, USDC)**.
- Melakukan top-up Rupiah melalui integrasi **Midtrans Sandbox** (QRIS, Virtual Account).
- Melakukan deposit crypto melalui **Blockchain Testnet** (Polygon Amoy / Sepolia).
- Mentransfer saldo sesama user di dalam platform.
- Menukar (swap) saldo antara Rupiah dan aset Crypto berdasarkan kurs real-time dari **Binance Public API**.
- Menarik (withdraw) saldo Rupiah dan Crypto ke rekening/wallet eksternal (simulasi via Sandbox/Testnet).

### 1.3 Definisi, Akronim, dan Singkatan

Lihat [Bagian 7: Glossary](#7-glossary).

### 1.4 Referensi

- [SWDD.md](./SWDD.md) — Software Design Description Document
- [Midtrans API Documentation](https://docs.midtrans.com/)
- [Polygon Amoy Testnet Explorer](https://amoy.polygonscan.com/)
- [Binance Public API](https://binance-docs.github.io/apidocs/spot/en/)
- [Alchemy SDK Docs](https://docs.alchemy.com/)
- Codebase eksisting: `github.com/bntngridp/ledger-backend-go`

---

## 2. Deskripsi Keseluruhan Sistem

### 2.1 Perspektif Sistem

Sistem Hybrid Wallet ini adalah **Centralized Custodial Wallet** yang menjadi jembatan antara ekosistem keuangan tradisional (perbankan & payment gateway) dengan ekosistem Crypto (blockchain). Sistem ini berperan sebagai:

- **Custodian** untuk aset crypto user (backend menyimpan private key/deposit address secara aman).
- **Ledger Internal** yang mencatat semua mutasi saldo dengan akurasi dan konsistensi tinggi.
- **Aggregator** yang menyatukan saldo dari berbagai jenis aset dalam satu akun user.

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT (Mobile/Web)                      │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTPS / REST API
┌────────────────────────────▼────────────────────────────────────┐
│                   API Server (Go / Gin)                         │
│                                                                 │
│  ┌─────────────────┐              ┌──────────────────────────┐  │
│  │  Fiat Engine    │              │     Crypto Engine        │  │
│  │  (IDR Module)   │              │  (EVM / Blockchain)      │  │
│  └────────┬────────┘              └───────────┬──────────────┘  │
└───────────│───────────────────────────────────│─────────────────┘
            │                                   │
            ▼                                   ▼
  ┌─────────────────┐                 ┌─────────────────────┐
  │ Midtrans Sandbox│                 │  Polygon Amoy       │
  │ (QRIS / VA)     │                 │  Testnet (EVM RPC)  │
  └─────────────────┘                 └─────────────────────┘
            │                                   │
            ▼                                   ▼
  ┌─────────────────┐                 ┌─────────────────────┐
  │  PostgreSQL DB  │                 │  Alchemy / Infura   │
  │  (Ledger Data)  │                 │  (Node Provider)    │
  └─────────────────┘                 └─────────────────────┘
                              ▲
                              │ Price Feed
                    ┌─────────────────────┐
                    │  Binance Public API │
                    │  (Exchange Rate)    │
                    └─────────────────────┘
```

### 2.2 Kelas Pengguna dan Karakteristik

| Kelas Pengguna | Deskripsi | Hak Akses |
|:---|:---|:---|
| **User (Authenticated)** | Pengguna terdaftar yang sudah login dan memiliki JWT Token yang valid. | Akses ke semua endpoint protected (top-up, transfer, swap, withdraw, history). |
| **Unauthenticated** | Pengguna yang belum login. | Hanya akses ke endpoint publik (register, login). |
| **Sistem Eksternal (Webhook)** | Midtrans Sandbox yang mengirim notifikasi pembayaran. | Akses ke endpoint webhook dengan verifikasi signature khusus. |

### 2.3 Lingkungan Operasional

- **Server**: Go 1.25+ berjalan di macOS (Development) / Linux (Production-like).
- **Database**: PostgreSQL 16.
- **Deployment Development**: Docker Compose (PostgreSQL via container).
- **Jaringan Crypto**: Polygon Amoy Testnet (Chain ID: 80002) / Sepolia Testnet.
- **Payment Gateway**: Midtrans Sandbox.
- **Price Feed**: Binance Public REST API (tidak butuh API Key).

---

## 3. Persyaratan Fungsional

### 3.1 Modul Autentikasi & Akun (AUTH)

#### FR-AUTH-001: Registrasi User Baru
- **Deskripsi**: Sistem harus memungkinkan user baru untuk mendaftar menggunakan username, email, dan password.
- **Input**: `username` (string, unik, max 50 char), `email` (format email valid, unik), `password` (min 6 char).
- **Proses**:
  1. Sistem memvalidasi format input.
  2. Sistem memeriksa keunikan email dan username di database.
  3. Sistem melakukan hash password menggunakan **bcrypt** (cost factor default).
  4. Sistem membuat record `User` dan `Wallet` secara **atomic** (satu database transaction).
  5. Wallet baru dibuat dengan saldo awal **0** untuk semua asset yang didukung (`IDR`, `USDT`, `USDC`).
- **Output**: Response sukses berisi `user_id`, `username`, `email`, `wallet_id`, dan list saldo awal.
- **Error Cases**: Email sudah terdaftar (`409 Conflict`), Username sudah dipakai (`409 Conflict`), Input tidak valid (`400 Bad Request`).

#### FR-AUTH-002: Login User
- **Deskripsi**: Sistem harus memungkinkan user terdaftar untuk login dan mendapatkan JWT Token.
- **Input**: `email`, `password`.
- **Proses**:
  1. Sistem mencari user berdasarkan email.
  2. Sistem memverifikasi password menggunakan `bcrypt.CompareHashAndPassword`.
  3. Sistem men-generate JWT Token yang berisi `user_id` dan waktu expiry.
- **Output**: Response berisi `token` (string JWT) dan `expires_in` (durasi dalam jam).
- **Error Cases**: Email tidak ditemukan (`401 Unauthorized`), Password salah (`401 Unauthorized`).

#### FR-AUTH-003: Dashboard Multi-Asset
- **Deskripsi**: Sistem harus menyediakan endpoint untuk mendapatkan ringkasan saldo semua aset milik user yang sedang login.
- **Input**: JWT Token di header `Authorization`.
- **Output**: 
  ```json
  {
    "wallet_id": "uuid",
    "balances": [
      { "asset": "IDR", "balance": 500000, "balance_display": "Rp 500.000" },
      { "asset": "USDT", "balance": 10.500000, "balance_display": "10.50 USDT" },
      { "asset": "USDC", "balance": 5.250000, "balance_display": "5.25 USDC" }
    ],
    "estimated_total_idr": 687250,
    "estimated_total_idr_display": "Rp 687.250"
  }
  ```

---

### 3.2 Modul Fiat / Rupiah (FIAT)

#### FR-FIAT-001: Inisiasi Top-Up Rupiah (Midtrans)
- **Deskripsi**: User dapat memulai proses top-up Rupiah dengan memilih metode pembayaran, dan sistem akan membuat invoice pembayaran via Midtrans Sandbox.
- **Input**: `amount` (int64, dalam Rupiah, minimum Rp 10.000), `payment_method` (`qris` | `bank_transfer`), opsional `bank_code` (jika `bank_transfer`).
- **Proses**:
  1. Sistem memvalidasi input.
  2. Sistem membuat record transaksi baru dengan status `pending` di database.
  3. Sistem memanggil **Midtrans Sandbox API** untuk membuat Charge transaksi.
  4. Midtrans mengembalikan data VA / QRIS String / QR Code URL.
  5. Sistem menyimpan `midtrans_order_id` dan detail pembayaran ke database.
- **Output**: Data pembayaran berisi `order_id`, `payment_type`, `va_number` (jika bank transfer), `qr_string` atau `qr_url` (jika QRIS), dan `expiry_time`.
- **Error Cases**: Amount di bawah minimum (`400`), Midtrans API error (`502 Bad Gateway`).

#### FR-FIAT-002: Menerima Konfirmasi Pembayaran Midtrans (Webhook)
- **Deskripsi**: Sistem harus menyediakan endpoint khusus untuk menerima notifikasi pembayaran dari Midtrans Sandbox.
- **Trigger**: HTTP POST dari Midtrans Sandbox ke endpoint `/api/v1/webhooks/midtrans`.
- **Proses**:
  1. Sistem memverifikasi **Signature Key** dari Midtrans menggunakan formula SHA512: `SHA512(order_id + status_code + gross_amount + server_key)`.
  2. Jika signature tidak cocok, request **ditolak** (`403 Forbidden`).
  3. Jika `transaction_status` adalah `settlement` atau `capture`, sistem mencari transaksi pending berdasarkan `order_id`.
  4. Sistem menambahkan saldo aset `IDR` user menggunakan **database transaction dengan pessimistic locking**.
  5. Sistem mengupdate status transaksi menjadi `success`.
- **Output**: HTTP 200 OK (diperlukan Midtrans untuk konfirmasi webhook diterima).
- **Error Cases**: Signature tidak valid (`403`), Transaksi tidak ditemukan (`404`), Transaksi sudah diproses (idempotent — kembalikan `200` tanpa proses duplikat).

#### FR-FIAT-003: Transfer Rupiah Antar User
- **Deskripsi**: User dapat mentransfer saldo Rupiah (`IDR`) ke user lain di dalam platform.
- **Input**: `destination_user_id` (UUID), `amount` (int64, minimum Rp 1.000), `notes` (string, opsional).
- **Proses**:
  1. Sistem memvalidasi user pengirim tidak mengirim ke dirinya sendiri.
  2. Sistem memeriksa saldo `IDR` pengirim mencukupi.
  3. Sistem melakukan debit-kredit saldo `IDR` secara **atomic** dengan **pessimistic locking** (`SELECT ... FOR UPDATE`).
  4. Sistem mencatat transaksi bertipe `transfer_fiat` dengan status `success`.
- **Output**: Detail transaksi berisi `transaction_id`, `amount`, `new_balance`, `recipient_username`.
- **Error Cases**: Saldo tidak cukup (`422 Unprocessable Entity`), Penerima tidak ditemukan (`404`), Transfer ke diri sendiri (`400`).

#### FR-FIAT-004: Tarik Saldo Rupiah ke Rekening Bank (Simulasi Disbursement)
- **Deskripsi**: User dapat mengajukan penarikan saldo Rupiah ke nomor rekening bank eksternal. Diproses menggunakan Midtrans Iris (Disbursement) Sandbox.
- **Input**: `amount` (int64, minimum Rp 50.000), `bank_code` (`bca`, `mandiri`, `bri`, dll), `account_number` (string), `account_name` (string).
- **Proses**:
  1. Sistem memvalidasi saldo `IDR` mencukupi termasuk biaya admin (simulasi Rp 2.500).
  2. Sistem mendebit saldo `IDR` user secara **atomic**.
  3. Sistem memanggil **Midtrans Iris Sandbox API** untuk mendisbursement dana.
  4. Sistem mencatat transaksi bertipe `withdraw_fiat` dengan status `pending`.
  5. Status diupdate ke `success` atau `failed` berdasarkan callback/polling Iris.
- **Output**: Detail withdrawal berisi `withdrawal_id`, `amount`, `bank`, `account_number`, `status`, `estimated_time`.
- **Error Cases**: Saldo tidak cukup (`422`), Kode bank tidak valid (`400`).

---

### 3.3 Modul Crypto (CRYPTO)

#### FR-CRYPTO-001: Generate Crypto Deposit Address
- **Deskripsi**: Sistem harus menyediakan setiap user sebuah deposit address unik untuk menerima kiriman aset crypto dari wallet eksternal (MetaMask, TrustWallet, dll).
- **Input**: JWT Token, `network` (`polygon_amoy` | `sepolia`), `asset` (`USDT` | `USDC`).
- **Proses**:
  1. Sistem memeriksa apakah user sudah memiliki deposit address untuk jaringan tersebut.
  2. Jika belum, sistem men-generate **EVM Address baru** menggunakan library `go-ethereum` (generate HD Wallet key pair dari entropy random yang aman).
  3. **Private key** dienkripsi menggunakan AES-256-GCM sebelum disimpan di database (TIDAK pernah disimpan plaintext).
  4. Sistem mengembalikan **hanya public address** kepada user.
- **Output**: `{ "address": "0xabc...123", "network": "polygon_amoy", "asset": "USDT", "qr_code_url": "..." }`.

#### FR-CRYPTO-002: Deteksi Deposit Crypto Masuk (On-Chain Listener)
- **Deskripsi**: Sistem harus secara otomatis mendeteksi transaksi USDT/USDC yang masuk ke deposit address milik user di blockchain testnet dan mengkredit saldo mereka.
- **Mekanisme**: Background **Goroutine** yang berjalan terus-menerus, memantau event `Transfer` dari smart contract USDT/USDC testnet menggunakan **Alchemy Websocket RPC**.
- **Proses saat event terdeteksi**:
  1. Sistem memverifikasi bahwa `to` address cocok dengan deposit address salah satu user di database.
  2. Sistem memverifikasi jumlah konfirmasi on-chain (minimum 5 block confirmations untuk keamanan).
  3. Sistem mengkredit saldo aset yang sesuai (USDT atau USDC) milik user secara **atomic**.
  4. Sistem mencatat transaksi bertipe `crypto_deposit` dengan `tx_hash` dari blockchain sebagai referensi.
  5. Sistem menerapkan **idempotency** menggunakan `tx_hash` — satu hash tidak boleh diproses lebih dari sekali.
- **Output**: Saldo user ter-update, notifikasi dapat dikirim (opsional — fitur masa depan).
- **Error Cases**: Duplikat `tx_hash` (di-skip), Jaringan RPC timeout (retry with backoff).

#### FR-CRYPTO-003: Kirim Crypto ke Wallet Eksternal (Withdrawal)
- **Deskripsi**: User dapat menarik aset crypto (USDT/USDC) dari saldo platform ke alamat wallet eksternal mereka.
- **Input**: `asset` (`USDT` | `USDC`), `network` (`polygon_amoy`), `to_address` (string, alamat EVM valid), `amount` (decimal, minimum 1 USDT/USDC).
- **Proses**:
  1. Sistem memvalidasi format `to_address` (harus berupa alamat EVM valid: 42 karakter, berawalan `0x`).
  2. Sistem memvalidasi saldo user mencukupi termasuk estimasi gas fee.
  3. Sistem mendebit saldo user secara **atomic** dan mencatat status `pending`.
  4. Sistem membangun dan menandatangani (*sign*) transaksi ERC-20 `transfer()` menggunakan private key deposit address user (di-dekripsi dari database sementara, lalu segera dihapus dari memori).
  5. Sistem mem-broadcast transaksi ke Polygon Amoy Testnet via Alchemy RPC.
  6. Sistem menyimpan `tx_hash` hasil broadcast.
  7. Background worker memantau konfirmasi on-chain dan mengupdate status transaksi ke `success` atau `failed`.
- **Output**: `{ "withdrawal_id": "uuid", "tx_hash": "0x...", "status": "pending", "amount": 10.0, "to_address": "0x..." }`.
- **Error Cases**: Saldo tidak cukup (`422`), Alamat tidak valid (`400`), Gas tidak cukup (`422`), RPC error (`502`).

---

### 3.4 Modul Exchange & Swap (EXCHANGE)

#### FR-EXCHANGE-001: Mendapatkan Kurs Real-time
- **Deskripsi**: Sistem harus menyediakan endpoint untuk mendapatkan kurs tukar terkini antara Rupiah dan aset crypto.
- **Input**: `pair` (contoh: `USDT_IDR`, `USDC_IDR`).
- **Proses**:
  1. Sistem mengambil harga dari **Binance Public API** (`GET /api/v3/ticker/price?symbol=USDTUSDT` dikombinasikan dengan kurs USD/IDR).
  2. Sistem melakukan **cache** hasil response selama 30 detik untuk menghindari rate-limiting.
- **Output**: `{ "pair": "USDT_IDR", "rate": 16200.50, "last_updated": "..." }`.

#### FR-EXCHANGE-002: Swap / Konversi Aset
- **Deskripsi**: User dapat langsung menukar saldo mereka antara Rupiah (IDR) dan Crypto (USDT/USDC), atau antar-Crypto.
- **Input**: `from_asset` (`IDR` | `USDT` | `USDC`), `to_asset` (`IDR` | `USDT` | `USDC`), `amount` (jumlah aset `from_asset` yang akan ditukar).
- **Proses**:
  1. Sistem mengambil kurs terbaru (dari cache atau refresh dari Binance API).
  2. Sistem menghitung jumlah aset `to_asset` yang akan diterima setelah dikurangi **biaya swap** (0.5% dari nilai tukar — simulasi biaya platform).
  3. Sistem menampilkan preview kepada user (berapa yang akan diterima).
  4. Sistem melakukan debit aset `from_asset` dan kredit aset `to_asset` secara **atomic** dalam satu database transaction.
  5. Sistem mencatat transaksi bertipe `swap` dengan detail rate dan fee yang digunakan.
- **Output**: Detail swap berisi `from_amount`, `to_amount`, `rate_used`, `fee_charged`, `new_balance_from`, `new_balance_to`.
- **Error Cases**: Saldo tidak cukup (`422`), Pair yang sama (`400`), Rate tidak tersedia (`503`).

---

### 3.5 Modul Riwayat Transaksi (HISTORY)

#### FR-HISTORY-001: Riwayat Transaksi Terpadu
- **Deskripsi**: Sistem harus menyediakan endpoint untuk melihat semua riwayat transaksi dari semua jenis aset dalam satu daftar terpadu.
- **Input**: JWT Token. Opsional: `asset` (filter berdasarkan aset), `type` (filter berdasarkan tipe), `page`, `per_page`.
- **Output**: List transaksi dengan pagination berisi:
  - `transaction_id`
  - `type` (`topup_fiat`, `transfer_fiat`, `withdraw_fiat`, `crypto_deposit`, `crypto_withdrawal`, `swap`)
  - `asset` (`IDR`, `USDT`, `USDC`)
  - `amount`
  - `status` (`pending`, `success`, `failed`)
  - `notes`
  - `tx_hash` (untuk transaksi crypto, nullable)
  - `created_at`
- **Pagination**: Default `per_page=20`, maksimum `per_page=100`.

---

## 4. Persyaratan Non-Fungsional

### 4.1 Keamanan (Security)

| Kode | Persyaratan |
|:---|:---|
| **NFR-SEC-001** | Semua password user harus di-hash menggunakan **bcrypt** dengan cost factor minimum 10. |
| **NFR-SEC-002** | Semua endpoint protected harus memvalidasi JWT Token di setiap request. JWT tidak boleh di-skip meskipun token masih terlihat valid di layer lain. |
| **NFR-SEC-003** | **Private key** crypto TIDAK BOLEH disimpan dalam bentuk plaintext di database. Wajib dienkripsi menggunakan **AES-256-GCM** dengan encryption key yang disimpan di environment variable (tidak di kode). |
| **NFR-SEC-004** | Endpoint webhook Midtrans WAJIB memverifikasi **Signature Key** SHA-512 sebelum memproses payload. Request tanpa signature valid harus ditolak dengan `403 Forbidden`. |
| **NFR-SEC-005** | Sistem harus menerapkan **idempotency** untuk setiap operasi keuangan. Satu Midtrans Order ID dan satu blockchain `tx_hash` hanya boleh diproses tepat satu kali. |
| **NFR-SEC-006** | Private key crypto yang di-dekripsi untuk keperluan signing transaksi harus segera di-zero-out dari memori setelah digunakan. |
| **NFR-SEC-007** | Tidak ada informasi sensitif (stack trace, query SQL, private key) yang boleh bocor ke response API yang diberikan kepada client. |

### 4.2 Keandalan & Konsistensi Data (Reliability)

| Kode | Persyaratan |
|:---|:---|
| **NFR-REL-001** | Semua operasi keuangan yang melibatkan perubahan saldo **wajib** menggunakan **database transaction** yang bersifat **ACID**. |
| **NFR-REL-002** | Semua operasi yang memodifikasi saldo dua pihak sekaligus (transfer, swap) **wajib** menggunakan **pessimistic locking** (`SELECT ... FOR UPDATE`) untuk mencegah race condition. |
| **NFR-REL-003** | Tipe data saldo untuk Rupiah menggunakan `BIGINT` (unit: sen atau Rupiah penuh). Tipe data saldo untuk aset Crypto menggunakan `DECIMAL(36, 18)` untuk menghindari floating-point error. |
| **NFR-REL-004** | Background worker On-Chain Listener harus memiliki mekanisme **retry dengan exponential backoff** jika koneksi ke Alchemy RPC terputus. |

### 4.3 Performa (Performance)

| Kode | Persyaratan |
|:---|:---|
| **NFR-PERF-001** | Response time untuk endpoint API standar (tidak termasuk panggilan ke Midtrans/Blockchain) tidak boleh melebihi **500ms** pada kondisi normal. |
| **NFR-PERF-002** | Data kurs dari Binance API harus di-**cache** selama minimum 30 detik untuk menghindari rate-limiting dan mempercepat response endpoint swap. |
| **NFR-PERF-003** | Endpoint list transaksi wajib mendukung **pagination** dengan default 20 item per halaman. |

### 4.4 Kemudahan Pemeliharaan (Maintainability)

| Kode | Persyaratan |
|:---|:---|
| **NFR-MAIN-001** | Kode harus mengikuti **Clean Architecture** dengan dependency flow yang ketat: `Handler → Usecase → Repository → Domain`. |
| **NFR-MAIN-002** | Setiap layer harus berkomunikasi melalui **interface** yang didefinisikan di package `domain`. |
| **NFR-MAIN-003** | Semua konfigurasi (API Key, secret, DSN) harus dibaca dari **environment variable** melalui file `.env`. Tidak ada nilai konfigurasi yang di-hardcode di kode sumber. |

---

## 5. Batasan Sistem

| No | Batasan |
|:---|:---|
| 1 | Sistem ini menggunakan **Midtrans Sandbox**, bukan Production. Tidak ada uang Rupiah sungguhan yang berpindah. |
| 2 | Sistem ini menggunakan **Polygon Amoy Testnet / Sepolia**, bukan Mainnet. Semua aset crypto bersifat token testnet tanpa nilai nyata. |
| 3 | Sistem **tidak** mendukung custodial private key untuk jaringan Bitcoin (UTXO model). Hanya jaringan EVM (Ethereum-compatible) yang didukung. |
| 4 | Sistem hanya mendukung aset stablecoin: **USDT** dan **USDC**. Tidak mendukung aset crypto yang volatil (BTC, ETH) dalam versi ini. |
| 5 | Fitur notifikasi real-time ke pengguna (push notification, WebSocket) **bukan** bagian dari scope versi ini. |
| 6 | Sistem **tidak** menyediakan fitur KYC (Know Your Customer) / verifikasi identitas pengguna pada versi ini. |

---

## 6. Asumsi dan Ketergantungan Eksternal

### 6.1 Asumsi

- Developer memiliki akun Midtrans Sandbox yang aktif dan sudah mendapatkan `Server Key` dan `Client Key`.
- Developer memiliki akun Alchemy (Free Tier) dan sudah mendapatkan `Alchemy API Key` untuk Polygon Amoy Testnet.
- Server berjalan di lingkungan yang memiliki akses internet untuk terhubung ke Midtrans, Alchemy, dan Binance API.
- Token testnet (USDT/USDC di Amoy) bisa didapatkan gratis dari Faucet masing-masing jaringan untuk keperluan testing.

### 6.2 Ketergantungan Eksternal

| Layanan Eksternal | Tujuan | Versi / Endpoint |
|:---|:---|:---|
| **Midtrans Sandbox** | Payment Gateway untuk top-up dan disbursement Rupiah | `https://api.sandbox.midtrans.com` |
| **Midtrans Iris Sandbox** | Disbursement / penarikan Rupiah | `https://app.sandbox.midtrans.com/iris` |
| **Alchemy** | EVM RPC Node untuk memantau transaksi blockchain | Polygon Amoy RPC & WebSocket |
| **Binance Public API** | Kurs harga real-time untuk fitur swap | `https://api.binance.com/api/v3` |
| **PostgreSQL 16** | Database utama untuk semua data ledger | `localhost:5432` (via Docker) |

---

## 7. Glossary

| Istilah | Definisi |
|:---|:---|
| **Fiat** | Mata uang yang diterbitkan oleh pemerintah, dalam konteks ini adalah **Rupiah (IDR)**. |
| **Stablecoin** | Aset crypto yang nilainya dipatok (dipeg) ke aset stabil seperti Dolar AS. Contoh: USDT, USDC. |
| **Custodial Wallet** | Sistem wallet di mana platform (bukan user) yang menyimpan dan mengontrol private key. User mempercayakan asetnya kepada platform. |
| **Pessimistic Locking** | Strategi penguncian database di mana baris data dikunci saat dibaca menggunakan `SELECT ... FOR UPDATE` untuk mencegah perubahan bersamaan dari proses lain. |
| **Atomic Transaction** | Serangkaian operasi database yang dieksekusi sebagai satu unit kerja. Semua berhasil, atau semua dibatalkan (rollback). |
| **Webhook** | Mekanisme di mana server eksternal (Midtrans) secara aktif mengirimkan notifikasi HTTP POST ke endpoint kita ketika suatu event terjadi (pembayaran berhasil). |
| **EVM** | Ethereum Virtual Machine — mesin eksekusi yang digunakan oleh jaringan Ethereum dan kompatibel seperti Polygon. |
| **Testnet** | Jaringan blockchain versi uji coba yang memiliki semua fitur blockchain sungguhan tetapi menggunakan token tanpa nilai nyata. |
| **HD Wallet** | Hierarchical Deterministic Wallet — metode generate banyak pasangan kunci (public/private key) dari satu seed phrase tunggal. |
| **tx_hash** | Transaction Hash — ID unik dari sebuah transaksi di blockchain. Bersifat permanen dan tidak bisa dipalsukan. |
| **Idempotency** | Properti suatu operasi di mana menjalankannya berkali-kali menghasilkan efek yang sama seperti menjalankannya satu kali. Penting untuk mencegah double-credit dari webhook. |
| **AES-256-GCM** | Algoritma enkripsi simetris yang sangat aman, digunakan untuk mengenkripsi private key sebelum disimpan di database. |
| **Disbursement** | Proses pengiriman/pencairan dana dari platform ke rekening bank eksternal milik user. |
| **Swap** | Pertukaran langsung antara satu jenis aset ke aset lain (contoh: IDR ke USDT) di dalam platform tanpa harus keluar ke bursa eksternal. |
