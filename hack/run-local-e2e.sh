#/bin/bash

# Setup the environment for running the e2e tests from your
# desktop.

set -e

parseCluster() {
    # These are all globals.
    net=$1
    subnet=$2
    zone=$3
    selfLink=$4
    net=$(echo ${net} | sed 's+.*networks/\([-a-z]*\).*$+\1+')
    subnet=$(echo ${subnet} | sed 's+.*subnetworks/\([-a-z]*\)$+\1+')
    project=$(echo ${selfLink} | sed 's+.*/projects/\([-a-z]*\)/.*+\1+')
}

parseInstance() {
    local name=$1
    local zone=$2
    # Globals.
    nodeTag=$(gcloud compute instances describe gke-test-default-pool-2b81ea3d-sw3l --zone us-west2-b --format='value(tags.items[0])')
}

clusterName="$1"
if [ -z ${clusterName} ]; then
    echo "Usage: $0 CLUSTER_NAME"
    exit 1
fi

fmt='value(networkConfig.network,networkConfig.subnetwork,zone,selfLink,name)'
parseCluster $(gcloud container clusters list --format=${fmt} | grep "${clusterName}\$")
parseInstance $(gcloud compute instances list --format='value(name,zone)' | grep ${clusterName} | tail -n 1)

gceConf="/tmp/gce.conf"
echo "Writing ${gceConf}"
echo "----"
cat <<EOF |  tee ${gceConf}
[global]
token-url = nil
project-id = ${project}
network-name = ${net}
subnetwork-name = ${subnet}
node-instance-prefix = ${clusterName}
node-tags = ${nodeTag}
local-zone = ${zone}
EOF

runScript="/tmp/run-glbc.sh"
echo "Writing ${runScript}". Use this to run the controller.
echo "----"
cat <<EOF | tee ${runScript}
#!/bin/bash

GOOGLE_APPLICATION_CREDENTIALS="${HOME}/.config/gcloud/application_default_credentials.json"

if [ ! -r ${GOOGLE_APPLICATION_CREDENTIALS} ]; then
    echo "You must login your application default credentials"
    echo "$ gcloud auth application-default login"
    exit 1
fi

GLBC=\${GLBC:-./glbc}
PORT=\${PORT:-7127}
V=\${V:-3}

echo "\$(date) start" >> /tmp/kubectl-proxy.log
kubectl proxy --port="\${PORT}" \\
    >> /tmp/kubectl-proxy.log &

PROXY_PID=\$!
cleanup() {
    echo "Killing proxy (pid=\${PROXY_PID})"
    kill \${PROXY_PID}
}
trap cleanup EXIT

kubectl apply -f docs/deploy/resources/default-http-backend.yaml

sleep 2 # Wait for proxy to start up
${GLBC} \\
    --apiserver-host=http://localhost:\${PORT} \\
    --running-in-cluster=false \\
    --logtostderr --v=\${V} \\
    --config-file-path=${gceConf} \\
    | tee -a /tmp/glbc.log
EOF

