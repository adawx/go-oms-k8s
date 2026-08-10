# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###

# Uncomment to use secrets
# k8s_yaml('./infra/development/k8s/secrets.yaml')

k8s_yaml('./infra/development/k8s/app-config.yaml')

### End of K8s Config ###

### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway'
if os.name == 'nt':
  gateway_compile_cmd = './infra/development/docker/api-gateway-build.bat'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")


docker_build_with_restart(
  'go-oms/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/api-gateway-deployment.yaml')
k8s_resource('api-gateway', port_forwards=8081,
             resource_deps=['api-gateway-compile'], labels="services")
### End of API Gateway ###

### Helm Remote Extension ###
load('ext://helm_remote', 'helm_remote')

### CloudNativePG Operator ###
helm_remote(
  'cloudnative-pg',
  repo_name='cnpg',
  repo_url='https://cloudnative-pg.github.io/charts',
  namespace='cnpg-system',
  create_namespace=True,
  version='0.29.0',
  set=[
    'webhook.mutating.failurePolicy=Ignore',
    'webhook.validating.failurePolicy=Ignore',
  ],
)
### End of CloudNativePG Operator ###

### Postgres (oms-db) ###
k8s_yaml('./infra/development/k8s/postgres.yaml')
k8s_resource(
  new_name='oms-db',
  objects=['oms-db:cluster'],       
  resource_deps=['cloudnative-pg'],      
  labels="databases"
)
### End of Postgres ###

### Observability: Prometheus + Grafana ###

# The prometheus-operator CRDs are installed outside Helm: helm_remote runs
# `helm template`, which does not render the chart's crds/ directory, and the
# CRDs are too large for a client-side apply. The script pins operator v0.93.0
# to match chart 88.2.0 below -- bump both together.
local_resource(
  'prometheus-crds',
  './infra/development/prometheus/install-crds.sh',
  deps=['./infra/development/prometheus/install-crds.sh'],
  labels="observability",
)

helm_remote(
  'kube-prometheus-stack',
  repo_name='prometheus-community',
  repo_url='https://prometheus-community.github.io/helm-charts',
  namespace='monitoring',
  create_namespace=True,
  version='88.2.0',
  set=[
    # CRDs are handled by the prometheus-crds resource above.
    'crds.enabled=false',

    # By default the operator only selects monitors labelled with this Helm
    # release. Our workloads live in `default`, so widen the selection to all
    # ServiceMonitors, PodMonitors and PrometheusRules in the cluster.
    'prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false',
    'prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false',
    'prometheus.prometheusSpec.ruleSelectorNilUsesHelmValues=false',

    # Keep local development light: no paging, short retention, no persistent
    # volumes to leak between `tilt down` cycles.
    'alertmanager.enabled=false',
    'prometheus.prometheusSpec.retention=6h',
    'prometheus.prometheusSpec.resources.requests.memory=400Mi',
    'grafana.persistence.enabled=false',

    # Local-only credentials.
    'grafana.adminPassword=admin',
    'grafana.defaultDashboardsTimezone=browser',
  ],
)

# The Grafana sidecar loads any ConfigMap carrying the grafana_dashboard
# label, so dashboards stay version controlled as JSON rather than being
# clicked together in the UI and lost on the next teardown. The ConfigMap is
# built from the JSON at load time so the dashboard file stays the single
# source of truth.
k8s_yaml(encode_yaml({
  'apiVersion': 'v1',
  'kind': 'ConfigMap',
  'metadata': {
    'name': 'oms-services-dashboard',
    'namespace': 'monitoring',
    'labels': {'grafana_dashboard': '1'},
  },
  'data': {
    # str() because read_file returns a Blob, which encode_yaml cannot encode.
    'oms-services.json': str(read_file('./infra/development/k8s/dashboards/oms-services.json')),
  },
}))
k8s_resource(
  new_name='grafana-dashboards',
  objects=['oms-services-dashboard:configmap'],
  resource_deps=['grafana'],
  labels="observability",
)

k8s_resource(
  'kube-prometheus-stack-operator',
  resource_deps=['prometheus-crds'],
  labels="observability",
)
# Prometheus itself is a CR, not a workload in the chart: the operator turns
# it into a StatefulSet after apply, so Tilt tracks the CR (as with oms-db)
# rather than a Deployment it never renders.
k8s_resource(
  new_name='prometheus',
  objects=['kube-prometheus-stack-prometheus:prometheus'],
  port_forwards='9090:9090',
  resource_deps=['kube-prometheus-stack-operator'],
  labels="observability",
)
k8s_resource(
  'kube-prometheus-stack-grafana',
  new_name='grafana',
  # Tilt forwards to the pod, not through the Service, so this targets the
  # Grafana container port (3000). The Service's port 80 is irrelevant here --
  # using it yields a connection that opens and then returns an empty reply.
  port_forwards='3000:3000',
  resource_deps=['prometheus-crds'],
  labels="observability",
)

# ServiceMonitors and alerting rules for the OMS services. Applied after the
# CRDs exist, otherwise the apply fails on unknown kinds.
k8s_yaml('./infra/development/k8s/monitoring.yaml')
# The two *-admin Services are claimed automatically by the api-gateway and
# order-service resources (Tilt assigns them by label), so only the monitor
# and rule objects need an explicit home here.
k8s_resource(
  new_name='oms-monitors',
  objects=[
    'oms-services:servicemonitor',
    'oms-service-rules:prometheusrule',
  ],
  resource_deps=['prometheus-crds'],
  labels="observability",
)
### End of Observability ###


### SERVICE ###
### Order Service ###

order_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/order-service ./services/order'
if os.name == 'nt':
  order_compile_cmd = './infra/development/docker/order-build.bat'

local_resource(
  'order-service-compile',
  order_compile_cmd,
  deps=['./services/order', './shared'], labels="compiles")

docker_build_with_restart(
  'go-oms/order-service',
  '.',
  entrypoint=['/app/build/order-service'],
  dockerfile='./infra/development/docker/order-service.Dockerfile',
  only=[
    './build/order-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/order-service-deployment.yaml')
k8s_resource('order-service', port_forwards=8085,
             resource_deps=['order-service-compile', 'oms-db'], labels="services")
### End of Order Service ###


