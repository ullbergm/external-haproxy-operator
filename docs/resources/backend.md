# Backend Custom Resource (CR) Usage Guide

This document explains how to use the `Backend` Custom Resource (CR) for the external-haproxy-operator. The `Backend` CR allows you to declaratively define HAProxy backend configurations in Kubernetes, which are then managed and synchronized by the operator.

## Example Backend CR

```yaml
apiVersion: external-haproxy-operator.ullberg.us/v1alpha1
kind: Backend
metadata:
  labels:
  name: backend-sample
spec:
  adv_check: httpchk
  balance:
    algorithm: leastconn
  http_check_list:
    - headers:
        - fmt: example.com
          name: Host
      method: GET
      type: send
      uri: /
  name: example
  servers:
    - address: 127.0.0.1
      name: srv1
      port: 443
    - address: 127.0.0.2
      name: srv2
      port: 443
    - port: 443
      valueFrom:
        serviceRef:
          name: router-internal-default
          namespace: openshift-ingress
```

## What This Produces

Given the above CR, the operator will generate a HAProxy backend configuration similar to:

```
backend example
  description managed-by=external-haproxy-controller
  balance leastconn
  option httpchk
  http-check send meth GET uri / hdr Host example.com
  server srv1 127.0.0.1:443
  server srv2 127.0.0.2:443
  server infra-1.openshift.internal.com 192.168.0.66:443
  server infra-0.openshift.internal.com 192.168.0.65:443
```

- **balance**: Sets the load balancing algorithm (here, `leastconn`).
- **option httpchk**: Enables HTTP health checks.
- **http-check send**: Configures a custom HTTP check (GET / with Host: example.com).
- **server**: Each server entry is generated from the `servers` list. Static servers use the provided address and name. Dynamic servers (with `valueFrom.serviceRef`) resolve to the endpoints of the referenced Kubernetes Service, with each endpoint becoming a server entry.

## Key Fields

- `spec.name`: The name of the backend (used in HAProxy config).
- `spec.balance.algorithm`: Load balancing algorithm (e.g., `leastconn`, `roundrobin`).
- `spec.adv_check`: Advanced health check method (e.g., `httpchk`).
- `spec.http_check_list`: List of HTTP checks to perform.
- `spec.servers`: List of backend servers. Each server can be static (with `address` and `name`) or dynamic (using `valueFrom.serviceRef` to reference a Kubernetes Service).

## Dynamic Servers

If a server uses `valueFrom.serviceRef`, the operator will resolve the endpoints of the referenced Service and create a server entry for each endpoint. This allows for dynamic scaling and service discovery.

## Status and Conditions

The operator updates the status of the Backend resource to reflect reconciliation progress, validation errors, or issues with referenced services/endpoints.

## See Also
- [api/v1alpha1/backend_types.go](../api/v1alpha1/backend_types.go) for CRD Go types and validation
- [internal/controller/backend_controller.go](../internal/controller/backend_controller.go) for reconciliation logic
