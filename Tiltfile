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

### CloudNativePG Operator ###
load('ext://helm_remote', 'helm_remote')
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


