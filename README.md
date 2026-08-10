# go-oms-k8s

OMS using Go, Kubernetes. 

### Prerequisites

- Go, Docker, Kubectl, Mininkube, Helm, Tilt


run `tilt up` and pray.


### Observability

Prometheus and Grafana come up with `tilt up`.
Grafana is at http://localhost:3000 (`admin` / `admin`), Prometheus at http://localhost:9090.

See [docs/OBSERVABILITY.md](docs/OBSERVABILITY.md) for how metrics are collected, where the
dashboards live, and which versions have to be bumped together.

### Commands that have been run

`minikube start --cni=calico`





