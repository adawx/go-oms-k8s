# go-oms-k8s

OMS using Go, Kubernetes. 

### Prerequisites

- Go, Docker, Kubectl, Mininkube, Helm, Tilt


run `tilt up` and pray.


### Observability

`tilt up` brings up Prometheus and Grafana alongside the services, under the `observability` label.

| What | Local URL | Notes |
| --- | --- | --- |
| Grafana | http://localhost:3000 | Login `admin` / `admin` |
| Prometheus | http://localhost:9090 | Targets, rules and ad-hoc PromQL |

The **OMS Services** dashboard covers request rate, error rate, latency percentiles, in-flight
requests, Go runtime stats and the order-service database connection pool.
The chart's own Kubernetes dashboards ship alongside it.

#### How it fits together

Each service exposes `/metrics`, `/healthz` and `/readyz` on a separate admin port (9090) so
they are not reachable through the public api-gateway LoadBalancer.
The shared instrumentation lives in `shared/metrics`, and `shared/httpserver` wires up both
listeners plus graceful shutdown.

Prometheus discovers those endpoints through the ServiceMonitor in
`infra/development/k8s/monitoring.yaml`, which matches the `monitoring: oms` label on the
per-service admin Services.
Postgres metrics come from CNPG's own PodMonitor, enabled via `monitoring.enablePodMonitor`
in `infra/development/k8s/postgres.yaml`.

Dashboards are provisioned as code: `infra/development/k8s/dashboards/*.json` is loaded into a
ConfigMap labelled `grafana_dashboard: "1"`, which the Grafana sidecar picks up automatically.
Edit the JSON and Tilt reapplies it — changes made by hand in the Grafana UI are lost on teardown.

#### Version pinning

The prometheus-operator CRDs are installed by `infra/development/prometheus/install-crds.sh`
rather than by Helm, because Tilt's `helm_remote` runs `helm template`, which does not render a
chart's `crds/` directory.
The script pins operator `v0.93.0` to match chart `88.2.0` in the `Tiltfile` — **bump both together**.

### Commands that have been run

`minikube start --cni=calico`





