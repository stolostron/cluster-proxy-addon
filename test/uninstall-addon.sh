#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}

cd cluster-proxy-addon-chart || {
    printf "cd failed, cluster-proxy-addon-chart does not exist"
    return 1
}

echo "############  Deploy cluster-proxy-addon"
make clean
if [ $? -ne 0 ]; then
    echo "############ Failed to deploy"
    exit 1
fi

echo "############  Removing cluster-proxy-addon-chart"
rm -rf cluster-proxy-addon-chart
