# Grafana Dashboard Generation

The repository provides Grafana dashboards as Go templates. Environment variables are inserted during generation, allowing dashboards to be configured without manual editing.

## Required Environment Variables

| Variable | Description |
|----------|-------------|
| `GREPTIMEDB_DATASOURCE_UID` | Grafana data source UID for GreptimeDB. |
| `POSTGRES_DATASOURCE_UID` | Grafana data source UID for the Sling PostgreSQL database. |

## Generate Dashboards

Render the dashboards into the `build/` directory:

```bash
export GREPTIMEDB_DATASOURCE_UID=greptime_uid
export POSTGRES_DATASOURCE_UID=postgres_uid
make dashboard
```

The command validates that the required variables are present and writes `grafana-dashboard.json` and `grafana_dashboard_sling.json` to `build/`.

