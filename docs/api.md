# API

## Health

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/health` | Public | Health check |

## Auth

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/auth/register/request-otp` | Public | Request register OTP |
| POST | `/api/auth/register/verify-otp` | Public | Verify register OTP and receive register token |
| POST | `/api/auth/register/complete` | Register token | Complete registration |
| POST | `/api/auth/login` | Public | Login |
| POST | `/api/auth/refresh` | Public | Refresh access token |
| POST | `/api/auth/logout` | Public | Revoke one refresh token |
| POST | `/api/auth/logout-all` | JWT | Revoke all refresh tokens for current user |
| POST | `/api/auth/link-email` | JWT | Link email to current user |

## PC Agents

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/pc-agents/pairing/start` | Public | PC-agent creates a backend-issued `pc_agent_id` and pairing code |
| GET | `/api/pc-agents/pairing/status?pairing_session_id=<id>&device_code=<code>` | Public | PC-agent checks pairing status and receives credential once confirmed |
| POST | `/api/pc-agents/verify` | Agent credential | PC-agent verifies with `pc_agent_id` and `agent_secret` |
| PATCH | `/api/pc-agents/me/protection` | Agent credential | PC-agent enables or disables its own protection mode |
| POST | `/api/pc-agents/pairing/confirm` | JWT | User confirms a pairing code |
| GET | `/api/pc-agents` | JWT | List current user's linked PC agents |
| DELETE | `/api/pc-agents/{pc_agent_id}` | JWT | Unlink a PC agent owned by current user |
| PATCH | `/api/pc-agents/{pc_agent_id}/protection` | JWT | Enable or disable protection for a PC agent |

### Protection Request

```json
{
  "pc_agent_id": "agent-id",
  "agent_secret": "agent-secret",
  "enabled": true
}
```

## Mobile Devices

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/mobile-devices` | JWT | Register or update current user's mobile device FCM token |

### Mobile Device Request

```json
{
  "fcm_token": "firebase-token",
  "platform": "android"
}
```

## Alerts

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/alerts` | JWT | List current user's alert history |
