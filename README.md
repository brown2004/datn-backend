# EMQX Broker

`compose.yaml` runs a single-node EMQX broker with persistent Docker volumes.

## Run

```bash
docker compose up -d
```

Dashboard: `http://localhost:18083`

## Why this setup

- Uses `emqx/emqx:6.2.0` instead of `latest` to keep deployments reproducible.
- Sets a stable node name with `hostname: broker.local` and `EMQX_NODE_NAME=emqx@broker.local`.
- Persists `/opt/emqx/data` and `/opt/emqx/log`, which EMQX documents as the directories to keep.
