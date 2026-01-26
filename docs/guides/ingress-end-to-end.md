# End-to-End Ingress Guide with KGateway

This guide walks you through setting up a complete ingress solution with KGateway, from deploying a sample application to configuring advanced routing and TLS termination. By the end of this guide, you'll have a fully functional ingress setup that you can build upon for production use.

## What You'll Build

In this guide, you'll create:
- A sample web application (httpbin) deployed to Kubernetes
- A basic Gateway and HTTPRoute for ingress traffic
- Path-based routing to demonstrate advanced routing capabilities
- TLS termination for secure HTTPS traffic
- Verification steps to ensure everything works correctly

## Prerequisites

Before starting, ensure you have:

- **Kubernetes cluster** (v1.26+) with kubectl access
- **KGateway installed** in your cluster
  - If not installed, see the [installation documentation](https://kgateway.dev/docs/installation/)
  - Verify installation: `kubectl get pods -n kgateway-system`
- **Gateway API CRDs** installed (usually included with KGateway installation)
  - Verify: `kubectl get crd gateways.gateway.networking.k8s.io`

## Step 1: Deploy Sample Application

First, let's deploy a simple test application that we'll expose through our ingress:

```yaml
# Save as httpbin-app.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: httpbin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  namespace: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
      - name: httpbin
        image: docker.io/mccutchen/go-httpbin:v2.6.0
        ports:
        - containerPort: 8080
        args:
        - "-port"
        - "8080"
        - "-max-duration"
        - "600s"
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
  namespace: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - name: http
    port: 8000
    targetPort: 8080
```

Apply the application:

```bash
kubectl apply -f httpbin-app.yaml
```

Verify the deployment:

```bash
kubectl get pods -n httpbin
kubectl get svc -n httpbin
```

## Step 2: Create Basic Gateway Configuration

Now let's create a Gateway resource that will serve as our ingress point:

```yaml
# Save as basic-gateway.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: my-gateway
  namespace: httpbin
spec:
  gatewayClassName: kgateway
  listeners:
  - name: http
    protocol: HTTP
    port: 8080
    allowedRoutes:
      namespaces:
        from: Same
```

Apply the Gateway:

```bash
kubectl apply -f basic-gateway.yaml
```

Check the Gateway status:

```bash
kubectl get gateway my-gateway -n httpbin -o yaml
```

The Gateway should show `Ready: True` in its status conditions.

## Step 3: Create Basic HTTPRoute

Create an HTTPRoute to connect your application to the Gateway:

```yaml
# Save as basic-route.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin-route
  namespace: httpbin
spec:
  parentRefs:
  - name: my-gateway
  hostnames:
  - "httpbin.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: "/"
    backendRefs:
    - name: httpbin
      port: 8000
```

Apply the HTTPRoute:

```bash
kubectl apply -f basic-route.yaml
```

Verify the route:

```bash
kubectl get httproute httpbin-route -n httpbin -o yaml
```

## Step 4: Test Basic Ingress

Get the Gateway's external IP or LoadBalancer endpoint:

```bash
# For LoadBalancer service
kubectl get svc -n kgateway-system

# For NodePort or port-forward testing
kubectl port-forward -n kgateway-system svc/kgateway-proxy 8080:8080
```

Test the ingress (adjust the URL based on your setup):

```bash
# If using port-forward
curl -H "Host: httpbin.example.com" http://localhost:8080/get

# If using LoadBalancer (replace EXTERNAL-IP)
curl -H "Host: httpbin.example.com" http://EXTERNAL-IP:8080/get
```

You should see a JSON response from httpbin showing request details.

## Step 5: Advanced Routing - Path-Based Routing

Let's add path-based routing to demonstrate more advanced capabilities. First, deploy a second application:

```yaml
# Save as nginx-app.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:stable
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: httpbin
spec:
  selector:
    app: nginx
  ports:
  - name: http
    port: 80
    targetPort: 80
```

Apply the nginx application:

```bash
kubectl apply -f nginx-app.yaml
```

Now update the HTTPRoute to include path-based routing:

```yaml
# Save as advanced-route.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin-route
  namespace: httpbin
spec:
  parentRefs:
  - name: my-gateway
  hostnames:
  - "httpbin.example.com"
  rules:
  # Route /api/* to httpbin
  - matches:
    - path:
        type: PathPrefix
        value: "/api"
    backendRefs:
    - name: httpbin
      port: 8000
  # Route /web/* to nginx
  - matches:
    - path:
        type: PathPrefix
        value: "/web"
    backendRefs:
    - name: nginx
      port: 80
  # Default route to httpbin
  - matches:
    - path:
        type: PathPrefix
        value: "/"
    backendRefs:
    - name: httpbin
      port: 8000
```

Apply the updated route:

```bash
kubectl apply -f advanced-route.yaml
```

Test the path-based routing:

```bash
# Test API path (should go to httpbin)
curl -H "Host: httpbin.example.com" http://localhost:8080/api/get

# Test web path (should go to nginx)
curl -H "Host: httpbin.example.com" http://localhost:8080/web/

# Test default path (should go to httpbin)
curl -H "Host: httpbin.example.com" http://localhost:8080/status/200
```

> **💡 Learn More**: For advanced routing features like header-based routing, traffic splitting, and request/response transformations, see the [Advanced Routing Documentation](https://kgateway.dev/docs/routing/).

## Step 6: TLS Termination

For production use, you'll want HTTPS. Let's add TLS termination to our Gateway.

First, create a self-signed certificate for testing:

```bash
# Create a self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=httpbin.example.com/O=httpbin.example.com"

# Create Kubernetes secret
kubectl create secret tls httpbin-tls \
  --cert=tls.crt --key=tls.key -n httpbin
```

Update the Gateway to include HTTPS listener:

```yaml
# Save as tls-gateway.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: my-gateway
  namespace: httpbin
spec:
  gatewayClassName: kgateway
  listeners:
  - name: http
    protocol: HTTP
    port: 8080
    allowedRoutes:
      namespaces:
        from: Same
  - name: https
    protocol: HTTPS
    port: 8443
    tls:
      mode: Terminate
      certificateRefs:
      - name: httpbin-tls
    allowedRoutes:
      namespaces:
        from: Same
```

Apply the updated Gateway:

```bash
kubectl apply -f tls-gateway.yaml
```

Update the HTTPRoute to work with both HTTP and HTTPS:

```yaml
# Save as tls-route.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin-route
  namespace: httpbin
spec:
  parentRefs:
  - name: my-gateway
    sectionName: http
  - name: my-gateway
    sectionName: https
  hostnames:
  - "httpbin.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: "/api"
    backendRefs:
    - name: httpbin
      port: 8000
  - matches:
    - path:
        type: PathPrefix
        value: "/web"
    backendRefs:
    - name: nginx
      port: 80
  - matches:
    - path:
        type: PathPrefix
        value: "/"
    backendRefs:
    - name: httpbin
      port: 8000
```

Apply the updated route:

```bash
kubectl apply -f tls-route.yaml
```

Test HTTPS (you may need to port-forward port 8443 as well):

```bash
# Port-forward HTTPS port
kubectl port-forward -n kgateway-system svc/kgateway-proxy 8443:8443

# Test HTTPS (use -k to ignore self-signed certificate warnings)
curl -k -H "Host: httpbin.example.com" https://localhost:8443/get
```

> **🔒 Production TLS**: For production environments, use certificates from a trusted CA or integrate with cert-manager for automatic certificate management. See the [TLS and Security Documentation](https://kgateway.dev/docs/security/) for detailed guidance.

## Step 7: Verification and Troubleshooting

### Verify Gateway Status

```bash
kubectl get gateway my-gateway -n httpbin -o yaml
```

Look for:
- `status.conditions` showing `Ready: True`
- `status.listeners` showing `Programmed: True` for each listener

### Verify HTTPRoute Status

```bash
kubectl get httproute httpbin-route -n httpbin -o yaml
```

Look for:
- `status.parents[].conditions` showing `Accepted: True` and `ResolvedRefs: True`

### Check KGateway Logs

```bash
kubectl logs -n kgateway-system -l app.kubernetes.io/name=kgateway -f
```

### Common Issues and Solutions

1. **Gateway not ready**: Check if KGateway is properly installed and running
2. **Route not accepted**: Verify the Gateway allows routes from the HTTPRoute's namespace
3. **503 Service Unavailable**: Check if backend services are running and endpoints are available
4. **TLS issues**: Verify certificate secret exists and is properly formatted

## Next Steps

Congratulations! You now have a fully functional ingress setup with KGateway. Here are some areas to explore next:

### Production Readiness
- **Monitoring and Observability**: Set up metrics, logging, and tracing - [Observability Guide](https://kgateway.dev/docs/observability/)
- **Security Policies**: Implement authentication, authorization, and rate limiting - [Security Documentation](https://kgateway.dev/docs/security/)
- **High Availability**: Configure multiple Gateway replicas and load balancing - [HA Setup Guide](https://kgateway.dev/docs/installation/ha/)

### Advanced Features
- **Traffic Management**: Implement canary deployments, traffic splitting, and circuit breakers - [Traffic Management](https://kgateway.dev/docs/traffic-management/)
- **API Gateway Features**: Add request/response transformation, caching, and API versioning - [API Gateway Guide](https://kgateway.dev/docs/api-gateway/)
- **Multi-tenancy**: Set up namespace isolation and tenant-specific policies - [Multi-tenancy Guide](https://kgateway.dev/docs/multi-tenancy/)

### Integration
- **CI/CD Integration**: Automate deployments with GitOps workflows - [GitOps Guide](https://kgateway.dev/docs/gitops/)
- **Service Mesh**: Integrate with Istio for advanced service-to-service communication - [Service Mesh Integration](https://kgateway.dev/docs/service-mesh/)
- **AI Gateway**: Explore AI-specific features for LLM and inference workloads - [AI Gateway Documentation](https://kgateway.dev/docs/ai-gateway/)

## Cleanup

To remove the resources created in this guide:

```bash
kubectl delete namespace httpbin
kubectl delete -f basic-gateway.yaml
rm tls.key tls.crt  # Remove certificate files
```

---

**Need Help?** Join the [KGateway community on Slack](https://kgateway.dev/slack/) or check out the [troubleshooting documentation](https://kgateway.dev/docs/troubleshooting/) for additional support.