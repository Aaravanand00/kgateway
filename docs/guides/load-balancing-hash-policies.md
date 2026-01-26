# Load Balancing with Hash Policies

This guide explains how to configure hash-based load balancing using `BackendConfigPolicy` in KGateway. Hash policies allow you to consistently route requests to the same backend based on request attributes like headers, cookies, or source IP.

## Overview

KGateway supports two hash-based load balancing algorithms:
- **RingHash**: Provides consistent hashing with configurable ring size
- **Maglev**: Google's Maglev consistent hashing algorithm

Both algorithms can be configured with hash policies that determine how requests are distributed across backend endpoints.

> **📝 Migration Note**: In KGateway v2.0, `hashPolicies` were moved from `TrafficPolicy` to `BackendConfigPolicy` to better align with their backend-specific nature. If you were previously using `hashPolicies` in `TrafficPolicy`, you'll need to migrate them to `BackendConfigPolicy`.

## Hash Policy Types

Hash policies determine which request attributes are used for consistent hashing:

### Header-based Hashing
Route requests based on HTTP header values:

```yaml
hashPolicies:
- header:
    name: "x-user-id"
  terminal: true
```

### Cookie-based Hashing
Route requests based on HTTP cookie values:

```yaml
hashPolicies:
- cookie:
    name: "session-id"
    path: "/api"
    ttl: 30m
    httpOnly: true
    secure: true
    sameSite: Strict
  terminal: true
```

### Source IP Hashing
Route requests based on client IP address:

```yaml
hashPolicies:
- sourceIP: {}
  terminal: false
```

## Configuration Examples

### RingHash Load Balancer

```yaml
apiVersion: gateway.kgateway.dev/v1alpha1
kind: BackendConfigPolicy
metadata:
  name: ringhash-policy
spec:
  targetRefs:
  - name: my-service
    group: ""
    kind: Service
  loadBalancer:
    ringHash:
      minimumRingSize: 1024
      maximumRingSize: 2048
      hashPolicies:
      # Primary: hash by user ID header
      - header:
          name: "x-user-id"
        terminal: true
      # Fallback: hash by session header
      - header:
          name: "x-session-id"
        terminal: false
```

### Maglev Load Balancer

```yaml
apiVersion: gateway.kgateway.dev/v1alpha1
kind: BackendConfigPolicy
metadata:
  name: maglev-policy
spec:
  targetRefs:
  - name: my-service
    group: ""
    kind: Service
  loadBalancer:
    maglev:
      hashPolicies:
      # Primary: hash by session cookie
      - cookie:
          name: "session-id"
          ttl: 1h
        terminal: true
      # Fallback: hash by source IP
      - sourceIP: {}
        terminal: false
```

## Hash Policy Evaluation

Hash policies are evaluated in order until a `terminal: true` policy matches or all policies are processed:

1. **Terminal Policy**: When `terminal: true`, evaluation stops if the policy can extract a value
2. **Non-terminal Policy**: When `terminal: false`, evaluation continues to the next policy regardless
3. **Fallback**: If no policies match, requests are distributed using round-robin

## Complete Example

Here's a complete example showing both RingHash and Maglev configurations:

```yaml
# Gateway
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: hash-lb-gateway
spec:
  gatewayClassName: kgateway
  listeners:
  - protocol: HTTP
    port: 8080
    name: http
---
# Service for RingHash
apiVersion: v1
kind: Service
metadata:
  name: api-service-ringhash
spec:
  selector:
    app: api-service
  ports:
  - port: 8080
    targetPort: 8080
---
# RingHash BackendConfigPolicy
apiVersion: gateway.kgateway.dev/v1alpha1
kind: BackendConfigPolicy
metadata:
  name: api-ringhash-policy
spec:
  targetRefs:
  - name: api-service-ringhash
    group: ""
    kind: Service
  loadBalancer:
    ringHash:
      minimumRingSize: 512
      maximumRingSize: 1024
      hashPolicies:
      - header:
          name: "x-tenant-id"
        terminal: true
      - cookie:
          name: "user-session"
        terminal: false
---
# HTTPRoute
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-route
spec:
  parentRefs:
  - name: hash-lb-gateway
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /api
    backendRefs:
    - name: api-service-ringhash
      port: 8080
```

## Testing Hash Policies

To verify hash policies are working correctly:

1. **Deploy the configuration**:
   ```bash
   kubectl apply -f your-config.yaml
   ```

2. **Send requests with different hash values**:
   ```bash
   # Requests with same user ID should go to same backend
   curl -H "x-user-id: user123" http://your-gateway/api
   curl -H "x-user-id: user123" http://your-gateway/api
   
   # Requests with different user ID should potentially go to different backend
   curl -H "x-user-id: user456" http://your-gateway/api
   ```

3. **Check backend distribution** by examining logs or metrics from your backend pods.

## Best Practices

1. **Choose appropriate hash attributes**: Use stable attributes that don't change frequently
2. **Use terminal policies wisely**: Set `terminal: true` for primary hash policies to avoid unnecessary processing
3. **Provide fallbacks**: Include non-terminal policies as fallbacks for better distribution
4. **Monitor distribution**: Use metrics to ensure requests are distributed evenly across backends
5. **Ring size tuning**: For RingHash, larger ring sizes provide better distribution but use more memory

## Migration from TrafficPolicy

If you were previously using `hashPolicies` in `TrafficPolicy`, migrate them to `BackendConfigPolicy`:

### Before (TrafficPolicy - deprecated)
```yaml
# ❌ This no longer works in v2.0
apiVersion: gateway.kgateway.dev/v1alpha1
kind: TrafficPolicy
metadata:
  name: my-policy
spec:
  # hashPolicies were here - NO LONGER SUPPORTED
```

### After (BackendConfigPolicy - correct)
```yaml
# ✅ Correct way in v2.0+
apiVersion: gateway.kgateway.dev/v1alpha1
kind: BackendConfigPolicy
metadata:
  name: my-policy
spec:
  targetRefs:
  - name: my-service
    group: ""
    kind: Service
  loadBalancer:
    ringHash:  # or maglev
      hashPolicies:
      - header:
          name: "x-user-id"
        terminal: true
```

## Troubleshooting

### Hash Policies Not Working
- Verify the `BackendConfigPolicy` is correctly targeting your service
- Check that hash attributes (headers, cookies) are present in requests
- Ensure the service has multiple healthy endpoints for distribution to be visible

### Uneven Distribution
- Consider using different hash attributes
- Adjust ring size for RingHash load balancer
- Add fallback hash policies for better coverage

### Policy Status
Check the `BackendConfigPolicy` status for any configuration errors:

```bash
kubectl get backendconfigpolicy my-policy -o yaml
```

## Related Documentation

- [BackendConfigPolicy API Reference](https://kgateway.dev/docs/api/backendconfigpolicy/)
- [Load Balancing Overview](https://kgateway.dev/docs/traffic-management/load-balancing/)
- [Traffic Management Guide](https://kgateway.dev/docs/traffic-management/)

---

For more examples, see the [example-backendconfigpolicy-hash-policies.yaml](../../examples/example-backendconfigpolicy-hash-policies.yaml) file in the examples directory.