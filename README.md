# DATN Backend

Backend Go cho he thong canh bao chong trom laptop su dung IoT.

## Kien Truc

Codebase di theo huong clean architecture don gian:

- `cmd/server`: entrypoint cua ung dung. File `main.go` chi load config, mo ket noi DB, khoi tao dependency va start HTTP server.
- `internal/config`: doc cau hinh tu environment variables va gan gia tri mac dinh cho local development.
- `internal/domain`: chua toan bo model/entity nghiep vu cua he thong, vi du `User`, `AuthOTP`, `RefreshToken`, `Alert`, `PCAgent`, `MobileDevice`, auth input/output. Cac tang khac khong tu tao business model rieng.
- `internal/usecase`: chua logic ung dung. Tang nay dieu phoi repository, token service va domain model de thuc hien use case nhu request OTP, verify OTP, complete register, login, refresh token, logout, link email.
- `internal/repo`: khai bao repository interface ma usecase can. Tang usecase phu thuoc interface o day thay vi phu thuoc truc tiep PostgreSQL.
- `internal/repo/postgres`: cai dat repository bang PostgreSQL. Tang nay chiu trach nhiem SQL query va map du lieu DB ve domain model.
- `internal/database`: khoi tao ket noi database va cau hinh connection pool.
- `internal/database/migrations`: SQL migration dung de tao/cap nhat schema database.
- `internal/delivery/http`: tang HTTP cua backend, gom handler, middleware va helper response. Tang nay chi decode request, goi usecase va encode response.
- `internal/delivery/http/router`: noi dang ky route HTTP. Auth routes va feature routes duoc tach thanh cac ham rieng de de mo rong.
- `internal/token`: tao, verify access token JWT va generate/hash refresh token.
- `internal/notification`: contract gui thong bao va mock sender cho local/test.
- `deploy`: ha tang local bang Docker Compose, hien co PostgreSQL va EMQX.

## API Hien Co

- `GET /health`
- `POST /api/auth/register/request-otp`
- `POST /api/auth/register/verify-otp`
- `POST /api/auth/register/complete` yeu cau registration access token tu buoc verify OTP
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `POST /api/auth/logout-all` yeu cau `Authorization: Bearer <access_token>`
- `POST /api/auth/link-email` yeu cau `Authorization: Bearer <access_token>`
- `POST /api/features/devices/mobile` yeu cau `Authorization: Bearer <access_token>`
- `POST /api/features/devices/pc-agents/link` yeu cau `Authorization: Bearer <access_token>`
- `GET /api/features/devices/pc-agents` yeu cau `Authorization: Bearer <access_token>`

## Register Flow

1. Goi `POST /api/auth/register/request-otp` voi body:

```json
{
  "phone_number": "0962143076"
}
```

2. Goi `POST /api/auth/register/verify-otp` voi body:

```json
{
  "phone_number": "0962143076",
  "otp": "123456"
}
```

Neu OTP hop le, API tra ve registration access token. Token nay chi dung cho buoc complete register, khong dung de goi API user thong thuong.

3. Goi `POST /api/auth/register/complete` voi header:

```text
Authorization: Bearer <registration_access_token>
```

Body:

```json
{
  "full_name": "Nguyen Van A",
  "password": "password123"
}
```

Neu thanh cong, API tao user va tra ve session gom `access_token` va `refresh_token`.

## Device Linking APIs

Dang ky thiet bi mobile de backend co the gui thong bao:

```http
POST /api/features/devices/mobile
Authorization: Bearer <access_token>
```

```json
{
  "fcm_token": "firebase-token",
  "platform": "android"
}
```

Lien ket PC agent voi user hien tai bang ma thiet bi:

```http
POST /api/features/devices/pc-agents/link
Authorization: Bearer <access_token>
```

```json
{
  "device_code": "PC-ABC-123",
  "device_name": "Laptop cua A",
  "os_type": "windows"
}
```

Lay danh sach PC agent da lien ket:

```http
GET /api/features/devices/pc-agents
Authorization: Bearer <access_token>
```

## Environment Variables

- `APP_PORT`: cong HTTP, mac dinh `8080`.
- `DATABASE_URL`: PostgreSQL connection string, mac dinh `postgres://postgres:postgres@localhost:5433/datn?sslmode=disable`.
- `JWT_SECRET`: secret ky JWT. Can doi khi chay production.
- `JWT_ISSUER`: issuer cua access token, mac dinh `datn-backend`.
- `JWT_AUDIENCE`: audience cua access token, mac dinh `datn-api`.
- `ACCESS_TOKEN_TTL`: thoi gian song access token theo dinh dang Go duration, mac dinh `15m`.
- `REFRESH_TOKEN_TTL`: thoi gian song refresh token theo dinh dang Go duration, mac dinh `720h`.

## Chay Local

Khoi dong ha tang local:

```bash
docker compose -f deploy/docker-compose.yml up -d
```

Chay migration:

```bash
goose -dir internal/database/migrations postgres "postgres://postgres:postgres@localhost:5433/datn?sslmode=disable" up
```

Chay backend:

```bash
go run ./cmd/server
```

Kiem tra health:

```bash
curl http://localhost:8080/health
```

## Test

```bash
go test ./...
```

## Nguyen Tac Code

- Domain model chi nam trong `internal/domain`.
- Handler khong chua business logic; handler chi validate/decode request co ban, goi usecase va tra response.
- Usecase khong import package PostgreSQL cu the; usecase chi noi chuyen voi repository interface.
- Repository implementation khong tra row/database model ra ngoai; tat ca du lieu di ra tang ngoai la domain model.
- HTTP middleware chi xu ly cross-cutting concern cua HTTP, vi du authentication.
