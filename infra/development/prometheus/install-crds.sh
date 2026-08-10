#!/usr/bin/env bash
#
# Installs the prometheus-operator CRDs required by kube-prometheus-stack.
#
# These are installed outside Helm because Tilt's helm_remote runs `helm
# template`, which does not render the chart's crds/ directory. The CRDs are
# also large enough that a client-side apply exceeds the 256KB
# last-applied-configuration annotation limit, so server-side apply is required.
#
# The operator version is pinned and must be kept in step with the chart
# version in the Tiltfile: chart 88.2.0 ships operator v0.93.0.
set -euo pipefail

OPERATOR_VERSION="v0.93.0"
BASE_URL="https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${OPERATOR_VERSION}/example/prometheus-operator-crd"

CRDS=(
  alertmanagerconfigs
  alertmanagers
  podmonitors
  probes
  prometheusagents
  prometheuses
  prometheusrules
  scrapeconfigs
  servicemonitors
  thanosrulers
)

echo "Installing prometheus-operator CRDs (${OPERATOR_VERSION})..."

for crd in "${CRDS[@]}"; do
  url="${BASE_URL}/monitoring.coreos.com_${crd}.yaml"
  echo "  -> ${crd}"
  # --force-conflicts lets us take ownership of fields from a previous
  # client-side apply or an older operator install.
  kubectl apply --server-side --force-conflicts -f "${url}"
done

echo "Waiting for CRDs to be established..."
for crd in "${CRDS[@]}"; do
  kubectl wait --for=condition=Established --timeout=60s \
    "crd/${crd}.monitoring.coreos.com"
done

echo "prometheus-operator CRDs ready."
