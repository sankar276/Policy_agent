#!/bin/bash

# Script to generate TLS certificates for Policy Agent webhook
# This is an alternative to using cert-manager

set -e

NAMESPACE="policy-agent"
SERVICE_NAME="policy-agent-webhook"
CERT_DIR="./certs"

echo "🔐 Generating TLS certificates for Policy Agent webhook..."

# Create certs directory
mkdir -p "${CERT_DIR}"

# Generate CA private key
openssl genrsa -out "${CERT_DIR}/ca.key" 2048

# Generate CA certificate
openssl req -x509 -new -nodes \
  -key "${CERT_DIR}/ca.key" \
  -subj "/CN=${SERVICE_NAME}-ca" \
  -days 3650 \
  -out "${CERT_DIR}/ca.crt"

echo "✓ Generated CA certificate"

# Generate server private key
openssl genrsa -out "${CERT_DIR}/tls.key" 2048

# Generate certificate signing request (CSR)
cat > "${CERT_DIR}/csr.conf" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.4 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
EOF

openssl req -new \
  -key "${CERT_DIR}/tls.key" \
  -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" \
  -out "${CERT_DIR}/server.csr" \
  -config "${CERT_DIR}/csr.conf"

echo "✓ Generated certificate signing request"

# Sign the server certificate
openssl x509 -req \
  -in "${CERT_DIR}/server.csr" \
  -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/tls.crt" \
  -days 365 \
  -extensions v3_req \
  -extfile "${CERT_DIR}/csr.conf"

echo "✓ Generated server certificate"

# Display certificate info
echo ""
echo "Certificate Information:"
openssl x509 -in "${CERT_DIR}/tls.crt" -noout -text | grep -A1 "Subject:"
openssl x509 -in "${CERT_DIR}/tls.crt" -noout -text | grep -A3 "Subject Alternative Name:"

echo ""
echo "✅ Certificates generated successfully!"
echo ""
echo "Next steps:"
echo "1. Create Kubernetes secret:"
echo "   kubectl create secret tls policy-agent-webhook-certs \\"
echo "     --namespace=${NAMESPACE} \\"
echo "     --cert=${CERT_DIR}/tls.crt \\"
echo "     --key=${CERT_DIR}/tls.key"
echo ""
echo "2. Get CA bundle for webhook configuration:"
echo "   CA_BUNDLE=\$(cat ${CERT_DIR}/ca.crt | base64 | tr -d '\\n')"
echo "   sed \"s/\\\${CA_BUNDLE}/\$CA_BUNDLE/\" webhook-config.yaml | kubectl apply -f -"
