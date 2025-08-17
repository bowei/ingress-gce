# NEG Binding \- Design proposal

Author: [Alan Grosskurth](mailto:grosskur@google.com)
Status: Draft
Last updated: Jul 15, 2025
Self-link: [go/neg-binding-design-proposal](http://goto.google.com/neg-binding-design-proposal)

# Objectives

1. Allow decoupling NEG **lifecycle management** from **endpoint management**.
2. Enable graphs of GCP resources that include NEGs to be **created and deleted
   in the correct order** without excessive errors, retries, and backoffs.

# Background

[Network endpoint groups](https://cloud.google.com/load-balancing/docs/negs)
(NEGs) are a key building block for container-native networking. Originally, GCP
load balancers directed traffic to target groups containing VMs. However, in GKE
clusters, each node has multiple IP addresses which correspond to distinct
containers. NEGs allow grouping arbitrary addresses (ip:port) and managing the
members of this grouping in an efficient manner.

GKE currently provides a **standalone NEG** feature. In this model, users add an
annotation to a Service object, which instructs the NEG controller to:

1. Create a set of zonal NEG resources (one per nodepool zone) for each service port.
2. Add and remove IPs of pods that match the service selector on an ongoing basis.
3. Delete the set of zonal NEG resources when the service is deleted.

While standalone NEGs were originally intended as an end-user feature, they have
been reused as a building block by controllers such as CSM and Gateway API,
which each create a graph of resources that include NEGs.

# Design ideas

The key observation is that today NEG controller performs two different
functions for NEGs:

* **Lifecycle management:** Creation and deletion of NetworkEndpointGroup resources.
* **Endpoint management:** Addition and removal of network endpoints to specific
  NetworkEndpointGroup resources.

The endpoint management feature is the core value that allows synchronizing the
endpoints in a set of NEGs with Kubernetes pod IPs that match a given service
selector. But it comes packaged with lifecycle management of the NEGs
themselves, without full context of the graph of resources that depend on the
NEGs, which causes the complications mentioned above.

We propose to create a new Custom Resource Definition (CRD) named
NetworkEndpointGroupBinding that provides endpoint management functionality
without any bundled lifecycle management functionality. Consider the following
example:

```
apiVersion: networking.gke.io/v1
kind: NetworkEndpointGroupBinding
metadata:
  name: productpage
  namespace: bookinfo
spec:
  serviceRef:
    name: productpage
    port: 80
  networkEndpointGroups:
  - project: grosskur-gke-dev
    location: us-central1-a
    name: productpage
  - project: grosskur-gke-dev
    location: us-central1-b
    name: productpage
  - project: grosskur-gke-dev
    location: us-central1-c
    name: productpage
```

When this object is created, NEG controller performs the following actions:

* Remove all endpoints in the NEGs referenced by the NetworkEndpointGroupBinding
  object (to start from a clean state).

* Read the Service object productpage in the same namespace to find its
  selector.

* Watch pods in the namespace that match the selector.

* When a matching pod is created or deleted, invoke the corresponding
  [attachNetworkEndpoint](https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/attachNetworkEndpoints)
  or
  [detachNetworkEndpoint](https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/detachNetworkEndpoints)
  method to keep the set of endpoints synchronized.

* If the referenced Service does not exist (or does not have the relevant
  service port), an error is reported in the status section using a status
  condition.

* If any of the referenced network endpoint groups do not exist, an error is
  reported in the status section using a status condition, and NEG controller
  synchronizes endpoints for any network endpoints groups that do exist.

Critically, when the NetworkEndpointGroupBinding object is deleted (or the
entire namespace is deleted), the NEG controller is able to finish its
processing quickly. It removes all endpoints from the NEG and stops managing it.

Note that it is not valid to have multiple NetworkEndpointGroupBindings
referencing the same NEG. The NEG controller needs to have exclusive control
over synchronizing the endpoints in the NEG to match a specific service
selector.

The assumption here is that the higher-level product (such as CSM) is creating
the necessary NetworkEndpointGroup resources, doing appropriate bookkeeping, and
cleaning up these resources as needed in the correct order, properly taking into
account dependencies.

Note that NetworkEndpointGroupBinding has broader applicability beyond use by
higher-level GCP products. It can be a versatile abstraction for customers who
need to manage their entire load balancer stack of resources using a tool like
Terraform, and only take advantage of NEG controller to synchronize endpoints.

# Caveats

## Changes to node zones and subnets

Lifecycle management for NetworkEndpointGroup resources is not as easy as it
looks on the surface. One of the main challenges is that the set of NEGs
associated with a Kubernetes Service can change over time as:

* Node zones are added or removed
* Additional node subnets are added or removed (see [Multi Subnet Clusters - NEG
  &
  Ingress](https://docs.google.com/document/d/1QUb7CxPqZ46AqUSHGGZ4A35T2LmbTJrzTvxgKeZCeRE/edit?resourcekey=0-U_G_n1nlFNYRqlZJz5nhfA&tab=t.0))

Since NEG is a **per-zone, per-subnet** resource, changes in the set of zones
and subnets associated with cluster nodes means the set of NEGs across the
cluster need to change, in order to have coverage of all active zones and
subnets. Otherwise, if NEG coverage is insufficient, newly-created pods may not
be able to be placed into a suitable NEG, or unnecessary NEGs may be created,
consuming quota.

Furthermore, an upcoming [GKE
Auto-IPAM](https://docs.google.com/document/d/1H9M9Du4wwwOBrekJc8j87bpyGk01GjEyKtiu9ver4wE/edit?resourcekey=0-aq0A35XPUU7aE6qrkMOhQQ&tab=t.0)
feature ([b/341245801](http://b/341245801)) is planned to dynamically add
subnets to clusters, triggering changes to node subnets.

## Preventing confused deputy problems

A naive implementation of NetworkEndpointGroupBinding would be vulnerable to the
[confused
deputy](https://docs.google.com/document/d/1xa6CbWZpg0Ym5aFV3qXqxzlYqurfHXYYf1lWxVQkPrw/edit?resourcekey=0-srBET5GGX8AV-PyqVBSqTw&tab=t.0)
problem because it would allow a user to bind a Service object in **any**
namespace of the cluster to **any** NEG in the GCP project.

This is problematic in a multi-tenant scenario. For example, suppose there is a
sensitive service in namespace payments that is bound to a NEG. If the name of
the NEG is known, a user with access only to namespace testing could bind the
NEG for the sensitive service to a testing service, resulting in traffic being
load balanced unexpectedly.

To guard against this, NEG controller should look for some required metadata in
the NetworkEndpointGroup GCP resource that indicates the following information
about the intended service that is allowed to bind the NEG:

* Cluster hash
* Namespace
* Service name
* Port

For example, consider a NEG that looks like this:

```
name: "k8s1-6fd26992-kube-system-metrics-server-443-ac4ddb95"
description: "{
  \"cluster-uid\": \"df4c4d67-d918-449c-9fb8-a16aa36545d3\",
  \"namespace\": \"kube-system\",
  \"service-name\":\"metrics-server\",
  \"port\":\"443\"
}"
networkEndpointType: "GCE_VM_IP_PORT"
zone: "https://www.googleapis.com/compute/v1/projects/test/zones/us-central1-c"
network: "https://www.googleapis.com/compute/v1/projects/test/global/networks/app"
subnetwork:
 "https://www.googleapis.com/compute/v1/projects/test/regions/us-central1/subnetworks/app"
```

## TargetGroupBinding

AWS introduced
[TargetGroupBinding](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.1/guide/targetgroupbinding/targetgroupbinding/)
to solve an analogous problem. [Target
Groups](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-target-groups.html)
are a similar abstraction to Network Endpoint Groups. However, Target Groups are
regional resources, so typically only a single TargetGroup is bound to a
Kubernetes Service.

AWS has a blog post that discusses various patterns that TargetGroupBinding
enables by giving the user complete control over the lifecycle of all load
balancer resources, while still relying on Kubernetes controller infrastructure
to automatically synchronize endpoints. See [Patterns for TargetGroupBinding
with AWS Load Balancer
Controller](https://aws.amazon.com/blogs/containers/patterns-for-targetgroupbinding-with-aws-load-balancer-controller/).

# Appendix: Borg controllers using NEG Binding

Given NEG Binding as a building block, suppose a Borg controller wants to offer
the following functionality: *When a Service is labeled by the customer with a
specific annotation, the controller needs to synchronize:*

* *A set of NEGs for each service port (per-zone, per-subnet)*
* *A BackendService for each service port, pointing at the appropriate NEGs*
* *A TcpRoute for each service port, pointing at the BackendService*

Using NEG Binding, the controller would behave as follows for various flows.

## Creation flow

1. Watch the cluster-scoped NodeTopology CR named default to discover the
   subnets and zones for the nodes of the cluster.
2. Watch Service objects labeled with the specific annotation. To reconcile:

   1. Create the appropriate set of NEGs for the node topology, along with a
      NEGBinding object to synchronize the NEGs with endpoints.
   2. Create the appropriate set of BackendServices for the ports, with each one
      pointing at the appropriate subset of NEGs for that port.
   3. Create the appropriate set of TcpRoutes for the ClusterIP \+ port
      combinations, with each one pointing at the appropriate BackendService.

For each action in step (2), the controller figures out the names of the GCP
resource, stores it in Spanner, and then kicks off the LRO to create the
resources. On the next reconcile, it queries compute.googleapis.com to see if
the LRO has completed, and if there were any errors. If the controller crashes
after storing the resource name in Spanner, but before kicking off the creation
LRO, it retries the creation.

## Update flows

* If the \`NodeTopology\` CR changes, to add zones or subnets, create new NEGs
  appropriately and update the NEGBinding to cover them. If zones or subnets are
  removed, update BackendService resources appropriately to no longer reference
  those NEGs, delete the NEGs, and update NEGBinding to no longer cover them.

* If a \`Service\` port is added, create a new set of NEGs, update NEGBinding,
  create a new BackendService, and a new TcpRoute. If a \`Service\` port is
  deleted, do the reverse, i.e., delete the TcpRoute, delete the BackendService,
  delete the NEGBinding, and delete the set of NEGs.

## Deletion flow

1. Suppose a Service object’s specific annotation is removed. To reconcile:
   1. Delete the associated TcpRoutes.
   2. Delete the associated BackendServices.
   3. Delete the NEGBinding object and associated NEGs.

For each step in (1), the controller also updates Spanner to remove the data
about the owned resources. Customers can also freely delete a Service or an
entire Namespace, and the controller will reconcile in the background, deleting
the GCP resources. The entire cluster can even be deleted. This is fine because
when a cluster no longer exists, the controller can delete the chain of all
associated GCP resources because it has stored this data in Spanner.
