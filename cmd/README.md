# Tinker

Tinker is a CLI tool which accepts Kubernetes style resource manifests as input and acts on the outside world to reconcile state.

It acts similarly to Kubernetes controllers but runs outside of clusters. This can be useful for bootstrapping scenarios and managing infrastructure which is outside of a Kubernetes cluster.

## Scenarios (IGNORE ME, THESE ARE WRONG)
- As a user, I want to deploy a set of resources, some of which may have interdependencies, and let the platform ensure the desired state matches my intent or inform me of the failure reason.
- As a user, I want to use a CLI to bootstrap initial resources upon which I may deploy other orchestration layers.
- As a user, I want to define a set of resources in a git repository and run the CLI in daemon mode to ensure the actual resources match the desired state in the repository.
