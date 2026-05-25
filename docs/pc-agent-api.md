# PC-agent API Contract

Base URL local:

```text
http://localhost:8080
```

PC-agent không dùng JWT user. PC-agent chỉ dùng:

- `pairing_session_id + device_code` trong lúc liên kết.
- `pc_agent_id + agent_secret` sau khi liên kết thành công.

## Luồng Chính

1. Agent chưa có config local thì gọi `POST /api/pc-agents/pairing/start`.
2. Backend tạo `pc_agent_id`; agent lưu local `pc_agent_id` và hiển thị `device_code` cho user.
3. Agent poll `GET /api/pc-agents/pairing/status`.
4. Khi status `confirmed`, backend trả `pc_agent_id + agent_secret` đúng lần đầu.
5. Agent lưu local `pc_agent_id + agent_secret`.
6. Lần sau khởi động, agent gọi `POST /api/pc-agents/verify`.

## 1. Start Pairing

```http
POST /api/pc-agents/pairing/start
Content-Type: application/json
```

Request:

```json
{
  "device_name": "LAPTOP-DUONG",
  "os_type": "windows"
}
```

Success `201`:

```json
{
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "pairing_session_id": "7e6f6c83-19ad-4ca2-a7d4-9d84c1c3d2ab",
  "device_code": "A7K9Q2",
  "expires_in": 600
}
```

Agent cần lưu tạm:

```json
{
  "pc_agent_id": "...",
  "pairing_session_id": "...",
  "device_code": "..."
}
```

Lưu ý:

- API này cấp trước `pc_agent_id` bằng `pairing_sessions.requested_pc_agent_id`; `pc_agents` chỉ được tạo khi user confirm để luôn có `user_id`.
- `requested_pc_agent_id` được đảm bảo unique bằng unique index trong database.
- `device_name` và `os_type` chỉ là metadata, không dùng làm định danh chính.
- `expires_in` là số giây còn lại, hiện tại khoảng `600`.

## 2. Check Pairing Status

```http
GET /api/pc-agents/pairing/status?pairing_session_id=<pairing_session_id>&device_code=<device_code>
```

Pending `200`:

```json
{
  "status": "pending"
}
```

Expired `200`:

```json
{
  "status": "expired"
}
```

Confirmed lần đầu `200`:

```json
{
  "status": "confirmed",
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "agent_secret": "random-secret"
}
```

Agent phải lưu local:

```json
{
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "agent_secret": "random-secret"
}
```

Confirmed nhưng credential đã cấp rồi `200`:

```json
{
  "status": "confirmed",
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "credential_issued": true
}
```

Nếu gặp response này mà agent chưa có local config thì không lấy lại được `agent_secret`; cần pairing lại từ đầu.

Error `400`:

```json
{
  "message": "Mã liên kết không hợp lệ"
}
```

Error `400`:

```json
{
  "message": "Phiên liên kết không hợp lệ"
}
```

## 3. Verify Agent

Gọi khi PC-agent khởi động lại hoặc cần xác thực lại. Không gọi heartbeat liên tục.

```http
POST /api/pc-agents/verify
Content-Type: application/json
```

Request:

```json
{
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "agent_secret": "random-secret"
}
```

Success `200`:

```json
{
  "message": "agent verified",
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "protection_status": "disabled"
}
```

`protection_status` có thể là:

```text
enabled
disabled
```

Agent dùng field này để biết chế độ cảnh báo hiện tại.

Invalid credential `401`:

```json
{
  "message": "Thông tin xác thực thiết bị không hợp lệ"
}
```

## 4. User Confirm Endpoint

Endpoint này do Flutter/app/web gọi, không phải PC-agent gọi. PC-agent chỉ cần biết sau khi user confirm thì status sẽ chuyển sang `confirmed`.

```http
POST /api/pc-agents/pairing/confirm
Authorization: Bearer <access_token>
Content-Type: application/json
```

Request:

```json
{
  "device_code": "A7K9Q2"
}
```

Success `201`:

```json
{
  "message": "Liên kết thiết bị thành công",
  "agent": {
    "id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
    "device_name": "LAPTOP-DUONG",
    "os_type": "windows",
    "agent_status": "offline",
    "protection_status": "disabled",
    "last_seen_at": null
  }
}
```

## PC-agent State Machine

```text
no_local_config
  -> start_pairing
  -> show_device_code
  -> poll_status
  -> confirmed_with_agent_secret
  -> save_local_config
  -> verify_on_startup
```

Nếu verify fail `401`:

```text
delete local config or mark invalid
start pairing again
```

Nếu pairing expired:

```text
start pairing again
```

## Suggested Local Config

```json
{
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "agent_secret": "random-secret",
  "device_name": "LAPTOP-DUONG",
  "os_type": "windows"
}
```

Không lưu `device_code` làm credential dài hạn. `device_code` chỉ dùng trong phiên pairing.
