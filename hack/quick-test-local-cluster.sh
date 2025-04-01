# this script is used to test the cluster-proxy-addon, by sending a curl request to list api of local-cluster

#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export NEW_SERVER=https://$(oc get route -n multicluster-engine cluster-proxy-addon-user -o=jsonpath='{.spec.host}')/local-cluster

# Create a service account
oc create serviceaccount cluster-proxy-admin

# Bind the service account to the cluster-admin role
oc adm policy add-cluster-role-to-user cluster-admin -z cluster-proxy-admin

# Get the token for the service account
ADMIN_TOKEN=$(oc create token cluster-proxy-admin)

# Verify the token is not empty
if [ -z "$ADMIN_TOKEN" ]; then
    echo "Failed to retrieve token for cluster-proxy-admin"
    exit 1
fi


curl -k -H "Authorization: Bearer ${ADMIN_TOKEN}" ${NEW_SERVER}/api/v1
