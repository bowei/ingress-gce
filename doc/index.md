# Documentation

## L7 LB controller

The L7 load balancer controller is the core component responsible for handling Kubernetes Ingress resources. It runs as part of the `glbc` binary and is instantiated in `cmd/glbc/main.go`. The primary logic resides in `pkg/controller/controller.go`.

### Control Flow

1.  **Watchers:** The controller watches for changes to `Ingress`, `Service`, `BackendConfig`, and `FrontendConfig` resources.
    *   Source: `NewLoadBalancerController` function in `pkg/controller/controller.go`
2.  **Queueing:** Changes to these resources trigger a sync operation, adding the affected Ingress key to a work queue (`ingQueue`).
    *   Source: `lbc.ingQueue.Enqueue(obj)` calls within the event handlers in `NewLoadBalancerController` in `pkg/controller/controller.go`
3.  **Sync Loop:** The `sync` function processes items from the queue.
    *   Source: `sync(key string)` function in `pkg/controller/controller.go`
4.  **Translation:** For each Ingress, the `Translator` converts the Kubernetes objects into a desired state for the Google Cloud L7 load balancer (`utils.GCEURLMap`).
    *   Source: `lbc.Translator.TranslateIngress(...)` call in `sync`, with the translator implementation in `pkg/translator/translator.go`
5.  **Resource Management:** The `l7Pool` ensures the necessary Google Cloud resources (URL maps, target proxies, forwarding rules, etc.) are created, updated, or deleted to match the desired state.
    *   Source: `lbc.l7Pool.Ensure(lb)` call in `SyncLoadBalancer`, with the pool implementation in `pkg/loadbalancers/l7s.go`
6.  **Backend Syncing:** The `backendSyncer` manages backend services and instance groups or NEGs. It creates health checks and attaches backends (instance groups or NEGs) to the backend services.
    *   Source: `lbc.backendSyncer.Sync(...)` call in `SyncBackends`, with the syncer implementation in `pkg/backends/syncer.go`
7.  **Status Update:** The Ingress resource status is updated with the load balancer's IP address and other relevant information.
    *   Source: `updateIngressStatus` function in `pkg/controller/controller.go`

## L4 LB controller

The L4 load balancer controller manages Kubernetes `Service` resources of `type: LoadBalancer`. It handles both internal (ILB) and external network (NetLB) load balancers. The controller is instantiated in `cmd/glbc/main.go`, with the core logic in `pkg/l4lb/l4.go`.

### Control Flow

1.  **Watchers:** The controller watches for changes to `Service` resources.
    *   Source: `NewL4Controller` function in `pkg/l4lb/l4.go`
2.  **Filtering:** It filters for services that are of `type: LoadBalancer` and have the appropriate annotations for either ILB (`cloud.google.com/load-balancer-type: "Internal"`) or are external.
    *   Source: `needsUpdate` function in `pkg/l4lb/l4.go`
3.  **Queueing:** Relevant services are added to a work queue (`serviceQueue`).
    *   Source: `l4c.serviceQueue.Enqueue(key)` call in `needsSync` in `pkg/l4lb/l4.go`
4.  **Sync Loop:** The `sync` function processes services from the queue.
    *   Source: `sync(key string)` function in `pkg/l4lb/l4.go`
5.  **Resource Management:** The `L4ServiceController` is responsible for creating and managing the necessary Google Cloud resources.
    *   Source: `EnsureL4Stack` function in `pkg/l4lb/l4.go`
    *   Resources managed: Forwarding Rules, Backend Services, Health Checks, Firewall rules.
6.  **Backend Management:** For L4 load balancers, the backends are the cluster nodes, which are managed as instance groups.
    *   Source: `backends.NewPool` and `healthchecksl4.NewL4HealthChecks` in `NewL4ServiceController` in `pkg/l4lb/l4.go`

## NEG controller

The Network Endpoint Group (NEG) controller manages NEGs for services that are annotated to use them. NEGs allow the load balancer to target pod IPs directly instead of nodes. The controller is instantiated in `cmd/glbc/main.go`, and its logic is in `pkg/neg/controller.go`.

### Control Flow

1.  **Watchers:** The controller watches `Service` and `EndpointSlice` resources.
    *   Source: `NewController` function in `pkg/neg/controller.go`
2.  **Filtering:** It looks for services with the `cloud.google.com/neg` annotation.
    *   Source: `annotations.FromService(service).NEGAnnotation()` check in `processService` in `pkg/neg/controller.go`
3.  **Queueing:** Services that require NEGs are added to the `serviceQueue`. Endpoint changes are added to the `endpointQueue`.
    *   Source: `enqueueService` and `enqueueEndpointSlice` functions in `pkg/neg/controller.go`
4.  **Sync Loop:**
    *   The `serviceWorker` processes services to determine which ports need NEGs. It creates or deletes `Syncer` objects to manage the NEGs for each service port.
    *   The `endpointWorker` processes endpoint changes for a given service.
    *   Source: `serviceWorker` and `endpointWorker` functions in `pkg/neg/controller.go`
5.  **Syncer Management:** The `SyncerManager` manages the lifecycle of `Syncer`s. Each `Syncer` is responsible for a single NEG.
    *   Source: `c.manager.EnsureSyncers(...)` call in `processService`, with the manager implementation in `pkg/neg/syncer_manager.go`
6.  **NEG Syncing:** The `Syncer` adds or removes endpoints (pod IP and port combinations) from its corresponding NEG in Google Cloud based on the `EndpointSlice` data.
    *   Source: `Sync` method of the `Syncer` struct in `pkg/neg/syncer.go`
7.  **Readiness Gates:** The controller also manages pod readiness gates, updating pod conditions to reflect whether they are considered healthy in the NEG and ready to receive traffic from the load balancer.
    *   Source: `c.reflector.SyncPod(pod)` call in the pod event handler, with the reflector implementation in `pkg/neg/readiness/reflector.go`
