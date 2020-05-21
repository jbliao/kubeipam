#!/bin/sh -xe

SERVICE_ACCOUNT_PATH=/var/run/secrets/kubernetes.io/serviceaccount
CNI_KUBECONFIG=/host/etc/cni/net.d/cccni.kubeconfig
KUBE_CA_FILE=${SERVICE_ACCOUNT_PATH}/ca.crt

install -m 755 /cccni /host/opt/cni/bin
cat > ${CNI_KUBECONFIG} << EOF
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: ${KUBERNETES_SERVICE_PROTOCOL:-https}://[${KUBERNETES_SERVICE_HOST}]:${KUBERNETES_SERVICE_PORT}
    certificate-authority-data: $(cat ${KUBE_CA_FILE} | base64 | tr -d '\n')
users:
- name: kubeipam
  user:
    token: "$(cat ${SERVICE_ACCOUNT_PATH}/token)"
contexts:
- name: default-context
  context:
    cluster: local
    user: kubeipam
current-context: default-context
EOF

echo "Success entering sleeping state..."
while true; do
  sleep 10000000
done