#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}

rm -rf cluster-proxy-addon-chart

echo "############  Cloning cluster-proxy-addon-chart"
git clone https://github.com/open-cluster-management/cluster-proxy-addon-chart.git


cd cluster-proxy-addon-chart || {
    printf "cd failed, cluster-proxy-addon-chart does not exist"
    return 1
}

echo "############  Deploy cluster-proxy-addon"
CLUSTER_BASE_DOMAIN=$($KUBECTL get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}')  
make deploy
if [ $? -ne 0 ]; then
    echo "############ Failed to deploy"
    exit 1
fi