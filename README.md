# DATN Backend

Backend Golang scaffold cho he thong canh bao chong trom laptop su dung IoT.

## Package structure

- `cmd/server`: entrypoint cua ung dung backend.
- `internal/config`: load cau hinh placeholder.
- `internal/domain`: entity va constant nghiep vu cot loi.
- `internal/usecase`: application use case, dieu phoi nghiep vu.
- `internal/repo`: repository interface.
- `internal/repo/postgres`: PostgreSQL repository placeholder cho tang ha tang.
- `internal/delivery/http`: HTTP router va handler placeholder.
- `internal/notification`: interface gui thong bao va mock sender.
- `deploy`: file docker compose cho ha tang local.

## Run

```bash
go mod tidy
go run ./cmd/server
curl http://localhost:8080/health
```
