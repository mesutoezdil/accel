# siltide, Prometheus and Grafana in one command

```sh
docker compose -f deploy/compose/docker-compose.yml up
```

- Grafana on <http://localhost:3000>, no login, with the dashboard already loaded
- Prometheus on <http://localhost:9090>
- siltide's own API and metrics on <http://localhost:9800>

The dashboard is provisioned from `grafana/dashboards/siltide.json`, so edits
made in the browser last until the container restarts. Export the JSON and
replace that file to keep one.

On a machine with no NVIDIA driver, drop the `NVIDIA_DRIVER_CAPABILITIES`
environment variable: the other vendors are read from `/sys` and need nothing
from the container runtime. `pid: host` is what lets siltide name the
processes holding a device; without it the container sees only its own.
