# Alert MQTT to FCM Flow

Backend can consume PC-agent alert messages from MQTT, persist them into `alerts`,
then send Firebase Cloud Messaging notifications to the user's registered mobile
devices.

## Environment

```env
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=datn-backend-alerts
MQTT_ALERT_TOPIC=pcapp/alert/+
MQTT_USERNAME=
MQTT_PASSWORD=

FIREBASE_SERVICE_ACCOUNT_FILE=D:\path\to\firebase-service-account.json
# or:
GOOGLE_APPLICATION_CREDENTIALS=D:\path\to\firebase-service-account.json
FCM_PROJECT_ID=
```

If `MQTT_BROKER` is empty, MQTT alert consuming is disabled.
If no Firebase service account file is configured, backend uses the mock
notification sender.

## MQTT Payload

PC-agent publishes to:

```text
pcapp/alert/{pc_agent_id}
```

Payload:

```json
{
  "pc_agent_id": "8e4d44b1-1e1c-4a2f-a339-ec1c4d5f22e2",
  "device_id": "DATN_STM32_001",
  "event_type": "motion_alert",
  "message": "detected event motion_alert",
  "timestamp": "2026-06-01T09:00:00Z"
}
```

Backend maps `motion_alert` to `motion_detected`, inserts a row into `alerts`,
finds `mobile_devices` by the PC agent owner, then sends one Firebase Cloud
Messaging HTTP v1 message per registered token.
