# Policy Agent Kubernetes Webhook

This directory contains Kubernetes manifests for deploying the Policy Agent admission webhook server.

## Overview

The Policy Agent webhook provides real-time policy validation and enforcement for Kubernetes resources. It validates resources before they are created or updated, ensuring compliance with your organization's policies.

## Prerequisites

- Kubernetes cluster (1.19+)
- `kubectl` configured
- cert-manager (optional, for automatic certificate management)
- Anthropic API key

## Quick Start

### Option 1: Using cert-manager (Recommended)

1. Install cert-manager if not already installed:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

2. Create the policy-agent namespace:
   ```bash
   kubectl apply -f namespace.yaml
   ```

3. Create the Anthropic API key secret:
   ```bash
   kubectl create secret generic policy-agent-secrets \
     --namespace=policy-agent \
     --from-literal=anthropic-api-key=YOUR_API_KEY
   ```

4. Create policy ConfigMaps:
   ```bash
   # Create config map from your policy files
   kubectl create configmap policy-agent-policies \
     --namespace=policy-agent \
     --from-file=../../../policies/

   # Create configuration
   kubectl create configmap policy-agent-config \
     --namespace=policy-agent \
     --from-file=../../../config/policy-agent.yaml
   ```

5. Deploy certificate issuer and certificate:
   ```bash
   kubectl apply -f certificate.yaml
   ```

6. Wait for certificate to be ready:
   ```bash
   kubectl wait --for=condition=ready certificate/policy-agent-webhook-cert \
     --namespace=policy-agent \
     --timeout=60s
   ```

7. Deploy RBAC, Service, and Deployment:
   ```bash
   kubectl apply -f rbac.yaml
   kubectl apply -f service.yaml
   kubectl apply -f deployment.yaml
   ```

8. Wait for webhook pods to be ready:
   ```bash
   kubectl wait --for=condition=ready pod \
     -l app=policy-agent-webhook \
     --namespace=policy-agent \
     --timeout=120s
   ```

9. Get the CA bundle and update webhook configuration:
   ```bash
   CA_BUNDLE=$(kubectl get secret policy-agent-webhook-certs \
     --namespace=policy-agent \
     -o jsonpath='{.data.ca\.crt}')

   # Update webhook-config.yaml with CA bundle
   sed "s/\${CA_BUNDLE}/$CA_BUNDLE/" webhook-config.yaml | kubectl apply -f -
   ```

### Option 2: Manual TLS Certificate

If you don't want to use cert-manager, you can generate certificates manually:

```bash
# Generate certificates
./generate-certs.sh

# Create secret
kubectl create secret tls policy-agent-webhook-certs \
  --namespace=policy-agent \
  --cert=certs/tls.crt \
  --key=certs/tls.key

# Get CA bundle
CA_BUNDLE=$(cat certs/ca.crt | base64 | tr -d '\n')

# Update and apply webhook configuration
sed "s/\${CA_BUNDLE}/$CA_BUNDLE/" webhook-config.yaml | kubectl apply -f -
```

## Verification

1. Check webhook pods are running:
   ```bash
   kubectl get pods -n policy-agent
   ```

2. Check webhook logs:
   ```bash
   kubectl logs -n policy-agent -l app=policy-agent-webhook -f
   ```

3. Test with a sample resource:
   ```bash
   # This should be rejected due to policy violations
   kubectl apply -f ../../examples/kafka/invalid-topic.yaml
   ```

## Configuration

### Namespace Selector

By default, the webhook validates resources in all namespaces except:
- `kube-system`
- `kube-public`
- `policy-agent`

To skip validation for a specific namespace, add the label:
```bash
kubectl label namespace my-namespace policy-agent/validation=skip
```

### Object Selector

To skip validation for a specific resource, add the label:
```yaml
metadata:
  labels:
    policy-agent/validation: skip
```

### Failure Policy

The webhook is configured with `failurePolicy: Fail` (fail closed). This means:
- If the webhook is unreachable, resource creation/updates will be blocked
- This prevents policy violations during webhook downtime

To change to fail-open mode (allow resources if webhook is down):
```yaml
failurePolicy: Ignore
```

## Supported Resources

The webhook currently validates:
- **Kafka**: KafkaTopic, KafkaUser, KafkaConnect, KafkaConnector (Strimzi)
- **Kubernetes**: Deployment, StatefulSet, DaemonSet, Pod, Service, ConfigMap
- **Flux CD**: Kustomization, GitRepository, HelmRepository, HelmRelease
- **ArgoCD**: Application, AppProject

## Monitoring

### Metrics

The webhook exposes metrics at:
- `/healthz` - Health check
- `/readyz` - Readiness check

### Prometheus Integration

Add ServiceMonitor for Prometheus:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: policy-agent-webhook
  namespace: policy-agent
spec:
  selector:
    matchLabels:
      app: policy-agent-webhook
  endpoints:
    - port: webhook
      path: /metrics
```

## Troubleshooting

### Webhook Not Working

1. Check pods are running:
   ```bash
   kubectl get pods -n policy-agent
   ```

2. Check webhook logs:
   ```bash
   kubectl logs -n policy-agent -l app=policy-agent-webhook
   ```

3. Verify webhook configuration:
   ```bash
   kubectl get validatingwebhookconfiguration policy-agent-webhook -o yaml
   ```

4. Check certificate:
   ```bash
   kubectl get secret policy-agent-webhook-certs -n policy-agent
   ```

### Certificate Issues

If using cert-manager:
```bash
# Check certificate status
kubectl describe certificate policy-agent-webhook-cert -n policy-agent

# Check certificate request
kubectl get certificaterequest -n policy-agent

# Check issuer
kubectl describe issuer policy-agent-ca-issuer -n policy-agent
```

### Webhook Blocking Everything

If the webhook is blocking all resources:
1. Check if the API key is configured correctly
2. Verify policies are loaded (check ConfigMap)
3. Temporarily change `failurePolicy` to `Ignore` to debug

## Updating Policies

To update policies without redeploying:
```bash
# Update policy ConfigMap
kubectl create configmap policy-agent-policies \
  --namespace=policy-agent \
  --from-file=../../../policies/ \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart webhook pods to reload policies
kubectl rollout restart deployment/policy-agent-webhook -n policy-agent
```

## Uninstallation

```bash
# Remove webhook configuration first
kubectl delete validatingwebhookconfiguration policy-agent-webhook

# Delete resources
kubectl delete -f deployment.yaml
kubectl delete -f service.yaml
kubectl delete -f rbac.yaml
kubectl delete -f certificate.yaml

# Delete namespace (this will delete all resources)
kubectl delete namespace policy-agent
```

## Security Considerations

1. **TLS**: Always use TLS for webhook communication
2. **RBAC**: The webhook has minimal permissions (read-only)
3. **Fail-Closed**: Configured to block on webhook failures (prevent bypass)
4. **Non-Root**: Webhook runs as non-root user (65534)
5. **Read-Only Root FS**: Container filesystem is read-only
6. **Drop Capabilities**: All Linux capabilities dropped

## High Availability

The deployment is configured with:
- 2 replicas for redundancy
- Pod anti-affinity to spread across nodes
- Resource limits to prevent resource exhaustion
- Liveness and readiness probes
- Graceful shutdown handling

## Performance

Expected latency:
- p50: <100ms
- p95: <200ms
- p99: <500ms

For high-traffic clusters, consider:
- Increasing replicas
- Adding HPA (Horizontal Pod Autoscaler)
- Adjusting resource limits

## Support

For issues and questions:
- GitHub: https://github.com/policy-agent/policy-agent
- Documentation: https://policy-agent.io/docs
