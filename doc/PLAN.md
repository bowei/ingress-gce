# NEG Binding Implementation Plan

This document outlines a high-level plan for implementing the Network Endpoint Group (NEG) Binding feature, as proposed in the design document. The implementation is divided into four phases:

## Phase 1: Scaffolding and CRD Definition

The first phase focuses on setting up the necessary boilerplate and defining the
new Custom Resource Definition (CRD).

### **Define the `NetworkEndpointGroupBinding` CRD:**

*   Do not use kubebuilder. Let's reuse the mechanisms that are in use for
    this repo: `hack/update-codegen.sh`.
*   Create the Go struct definitions for the `NetworkEndpointGroupBinding`
    resource, including `spec` and `status` fields as described in the design
    document.
    *   **Source:** Create a new file, e.g.,
            `pkg/apis/negbinding/v1/types.go`, following the pattern of existing
            CRDs like `pkg/apis/svcneg/v1beta1/types.go`.

Schema:
*   The `spec` should include `serviceRef` and a list of
   `networkEndpointGroups`.
*   The `status` should include conditions to report errors, such as the
    referenced `Service` or NEGs not existing.
*   Generate the CRD manifest file using `controller-gen`.
*   **Source:** This will be invoked via the `hack/update-codegen2.sh` command.

*  **Scaffold the New Controller:**
    *   Create a new controller within the existing NEG controller framework to
        manage `NetworkEndpointGroupBinding` resources.
        *   **Source:** Create a new file, e.g.,
            `pkg/negbinding/controller.go`.
    *   Set up the necessary watchers for `NetworkEndpointGroupBinding`
        * Watch the NetworkEndpointGroupBinding resource.
        * Watch the Service resource.
    *   Initialize the required clients and informers.
*  **Testing the New Controller:**
    * Code build compile.
    * Create unit tests with Fake clients.
      * Add a simple test case that creates the resource, updates it, and deletes it.

## Phase 2: Core Controller Logic

This phase involves implementing the main reconciliation logic for the `NetworkEndpointGroupBinding` controller.

1.  **Implement the Reconciliation Loop:**
    * Update/Create
    
    *   The controller's main reconciliation loop will be triggered by changes
        to `NetworkEndpointGroupBinding` resources.
        *   **Source:** A new `sync` or `reconcile` function in
            `pkg/negbinding/controller.go`.
    *   When a `NetworkEndpointGroupBinding` is created or updated, the controller will:
        *   Fetch the referenced `Service` to get its selector.
        *   Add a NEG syncer to begin sync'ing the endpoints associated with the binding.
        *   TODO: We will have to be careful here. 
        *   **Source:** The reconciliation loop will likely reuse components from `pkg/neg/syncer_manager.go` and `pkg/neg/syncer.go`. The logic for calculating endpoints can be adapted from `pkg/neg/syncers/endpoints_calculator.go`.

    * Delete
        * Stop the NEG syncer.
        
3.  **Status Updates:**
    *   The controller will update the `status` of the
        `NetworkEndpointGroupBinding` resource with conditions to reflect its
        current state.
    *   This includes reporting errors if the referenced `Service` or any of the
        NEGs do not exist.
        *   **Source:** The reconciliation loop will use a generated clientset
            for the new CRD to call `UpdateStatus`.





## Phase 3: Security Implementation

This phase focuses on addressing the "confused deputy" problem described in the design document.

1.  **Implement Metadata Validation:**
    *   When a `NetworkEndpointGroupBinding` is reconciled, the controller will inspect the metadata of each referenced NEG in GCP.
        *   **Source:** Inside the reconciliation loop in `pkg/neg/binding_controller.go`.
    *   It will verify that the NEG's description or labels contain the required information to authorize the binding: Cluster hash/UID, Namespace, Service name, Port.
        *   **Source:** The controller will call `cloud.GetNetworkEndpointGroup` and parse the description field. The cluster UID is available from the `ControllerContext` (`ctx.KubeSystemUID`).
    *   If the metadata does not match the `serviceRef` in the `NetworkEndpointGroupBinding`, the controller will report an error in the status and will not manage the NEG.

2.  **Update Higher-Level Controllers:**
    *   The controllers that create the NEGs (e.g., CSM, Gateway API) will need to be updated to add the required metadata to the NEGs they create. This is outside the scope of the NEG controller itself but is a necessary part of the overall feature.

## Phase 4: Integration and Testing

The final phase involves integrating the new controller with the existing infrastructure and ensuring it is well-tested.

1.  **Integrate with Existing NEG Controller:**
    *   The new `NetworkEndpointGroupBinding` controller will run alongside the existing standalone NEG controller.
        *   **Source:** The new controller will be started in the `main` function in `cmd/glbc/main.go`.
    *   Care must be taken to ensure that a single NEG is not managed by both controllers simultaneously. The design document states that it is not valid to have multiple bindings for the same NEG, and this should be enforced.
        *   **Source:** The `SyncerManager` in `pkg/neg/syncer_manager.go` could be enhanced to track the owner of a NEG syncer (e.g., Service annotation vs. NEGBinding). Alternatively, a validating admission webhook could be implemented to prevent the creation of conflicting `NetworkEndpointGroupBinding` resources.

2.  **Develop a Testing Strategy:**
    *   **Unit Tests:** Create unit tests for the new controller's reconciliation logic, including error handling and status updates.
        *   **Source:** A new file, `pkg/neg/binding_controller_test.go`.
    *   **Integration Tests:** Develop integration tests that run against a real Kubernetes cluster and GCP environment to verify the end-to-end functionality.
    *   **E2E Tests:** Add end-to-end tests to cover the user-facing scenarios, including creating and deleting `NetworkEndpointGroupBinding` resources and verifying that the NEGs are correctly populated.
        *   **Source:** A new test file, e.g., `cmd/e2e-test/negbinding_test.go`, following the pattern of `cmd/e2e-test/neg_test.go`.
