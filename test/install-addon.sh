#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}

rm -rf cluster-proxy

echo "############  Apply CRD"
$KUBECTL apply -f https://raw.githubusercontent.com/stolostron/cluster-proxy/main/charts/cluster-proxy/crds/managedproxyconfigurations.yaml

rm -rf cluster-proxy-addon-chart

echo "############  Cloning cluster-proxy-addon-chart"
git clone https://github.com/stolostron/cluster-proxy-addon-chart.git


cd cluster-proxy-addon-chart || {
    printf "cd failed, cluster-proxy-addon-chart does not exist"
    return 1
}

echo "############  Deploy cluster-proxy-addon"
export CLUSTER_BASE_DOMAIN=$($KUBECTL get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}') && \
export IMAGE_CLUSTER_PROXY_ADDON=$1 && \
echo $CLUSTER_BASE_DOMAIN && echo $IMAGE_CLUSTER_PROXY_ADDON && make -e deploy

if [ $? -ne 0 ]; then
    echo "############ Failed to deploy"
    exit 1
fi
