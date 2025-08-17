# NEG Controller Architecture

This document provides a detailed overview of the Network Endpoint Group (NEG) Controller's architecture within the `ingress-gce` repository.

## Overview

The NEG Controller is responsible for creating and managing Google Cloud Network Endpoint Groups (NEGs) for Kubernetes services. NEGs are used as backends for both L7 and L4 load balancers, allowing the load balancer to send traffic directly to pod IPs rather than to cluster nodes. This enables advanced features like container-native load balancing.

The controller operates by watching Kubernetes API resources like `Service`, `Pod`, `EndpointSlice`, and `Node`. Based on annotations and the state of these resources, it determines which NEGs are needed and syncs the correct set of network endpoints (pod IP and port combinations) to them.

## File Organization

The core components of the NEG controller are organized as follows:

- **`cmd/glbc/main.go`**: The main entry point for the controller binary. It initializes and starts the `neg.Controller`. The NEG controller often runs with its own leader election (`ingress-gce-neg-lock`) to allow it to scale independently of the main ingress controller.

- **`pkg/neg/controller.go`**: This file contains the `Controller` struct, which is the core of the NEG controller. It includes event handlers for `Service`, `EndpointSlice`, `Pod`, and `Node` resources and orchestrates the NEG lifecycle via several work queues.

- **`pkg/neg/manager.go`**: The `syncerManager` is responsible for creating, deleting, and managing the lifecycle of individual NEG syncers. When the main controller determines that a service needs a NEG, it instructs the `syncerManager` to start a syncer for it. It is also responsible for garbage collecting unused NEGs.

- **`pkg/neg/syncers/`**: This directory contains the logic for the different types of NEGs. A "syncer" is responsible for a single NEG and manages the endpoints within it.
    - **`syncer.go`**: The main syncer loop that watches for changes in endpoints and triggers a sync. It manages the state of a syncer (running, stopped) and handles backoff retries.
    - **`transaction.go`**: Implements the `transactionSyncer`, which contains the core logic for syncing a NEG. It calculates the difference between the desired and actual endpoints and uses batched API calls to add and remove endpoints from the NEG.

- **`pkg/neg/readiness/`**: This directory contains the "readiness reflector," which is responsible for implementing the NEG readiness gate on pods.
    - **`reflector.go`**: The `readinessReflector` checks the health of endpoints as reported by the GCE health check and updates the `cloud.google.com/neg-readiness-gate` condition on the corresponding pods. This allows Kubernetes to know when a pod is considered healthy by the load balancer and ready to receive traffic.

## High-Level Workflow & Control Flow

The NEG controller's operation is event-driven, revolving around several work queues that process changes to Kubernetes resources.

1.  **Event Trigger**:
    - An event occurs for a `Service`, `Ingress`, `EndpointSlice`, or `Node` resource.
    - The corresponding event handler in `controller.go` enqueues a key into one of the work queues: `serviceQueue`, `endpointQueue`, or `nodeQueue`.

2.  **Service Processing (`serviceWorker`)**:
    - The `serviceWorker` dequeues a service key.
    - It evaluates the `Service` to determine if NEGs are required. This is based on:
        - The `cloud.google.com/neg` annotation for standalone NEGs or NEGs for L7 XLB.
        - Usage by an L7 ILB or L7 Regional XLB Ingress (which implicitly require NEGs).
        - Usage by an L4 ILB Service (which also implicitly requires NEGs).
    - It constructs a `PortInfoMap`, which represents the desired state of all NEGs for that service.

3.  **Syncer Management (`EnsureSyncers`)**:
    - The controller passes the desired `PortInfoMap` to the `syncerManager`.
    - The `syncerManager` compares the new `PortInfoMap` with its existing state for that service.
        - It stops syncers for NEGs that are no longer needed.
        - It starts new syncers for newly required NEGs.
    - For each new syncer, the `syncerManager`:
        - Creates a GCE NEG resource via the cloud provider if it doesn't already exist.
        - If the `ServiceNetworkEndpointGroup` CRD is enabled, it also creates a corresponding CR to track the NEG's lifecycle.
        - Starts a dedicated "syncer" goroutine for the NEG.

4.  **Endpoint Syncing (`endpointWorker` and `transactionSyncer`)**:
    - A change to an `EndpointSlice` resource triggers the `endpointWorker`.
    - The `endpointWorker` identifies the relevant service and signals the corresponding syncers to start a sync operation.
    - The `transactionSyncer.sync()` method is called:
        - It calculates the desired set of endpoints for the NEG using an `EndpointCalculator`.
        - It lists the current endpoints in the GCE NEG.
        - It computes the difference between the desired and current endpoints.
        - It uses batched API calls (`AttachNetworkEndpoints` and `DetachNetworkEndpoints`) to add and remove the necessary endpoints from the NEG.

5.  **Status Annotation**:
    - After determining the required NEGs, the `serviceWorker` updates the `cloud.google.com/neg-status` annotation on the `Service` object.
    - This annotation contains a JSON payload describing the name and zones of the NEGs it manages. This information is crucial for the L7 and L4 ingress controllers to find the correct NEGs to attach to their backend services.

6.  **Pod Readiness Gate (`readinessReflector`)**:
    - If a NEG is configured with a readiness gate, the `readinessReflector` takes over.
    - The `transactionSyncer` commits the list of pods in the NEG to the reflector.
    - The reflector's `poller` periodically polls the health status of endpoints in the NEG from the GCE health check service.
    - When an endpoint becomes healthy, the reflector patches the corresponding `Pod` resource, setting the `cloud.google.com/neg-readiness-gate` condition to `True`. The Kubelet sees this and allows the pod to proceed to the `Ready` state.

7.  **Garbage Collection**:
    - The controller periodically runs a garbage collection loop (`gc()`) to find and delete any GCE NEG resources that are no longer required by any Kubernetes service.
    - If the `ServiceNetworkEndpointGroup` CRD is enabled, GC is based on the presence of these CRs. If a CR exists but is not found in the controller's desired state map (`svcPortMap`), it is a candidate for deletion.
    - If the CRD is not used, the controller lists all NEGs in GCE with the expected naming convention and deletes any that are not in its desired state map.