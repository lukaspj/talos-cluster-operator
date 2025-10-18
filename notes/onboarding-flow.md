## Bootstrapping a node
Use `talos.config` to point to this operator's config endpoint. (Ideally with [auth](https://docs.siderolabs.com/talos/v1.11/security/machine-config-oauth)).

Node gets flashed and connects to the config endpoint which joins it as a node in the cluster.
When the config endpoint is called, it will create a corresponding Machine resource.

The node is now ready to be used.

## Creating a new cluster
A new Cluster resource is created, with a Machine selector that chooses which machines should form 
the control plane and a different selector for the worker nodes.

First, one of the control plane machines is selected, we apply a new Talos config which instructs it 
to drop out of the management cluster and create a new cluster. The cluster is bootstrapped. After 
bootstrapping has finished, the other control plane machines are updated to join the new cluster.

Then all worker nodes are updated to join the new cluster.

## Choosing Machines
A Machine is only available to join a cluster if it is currently part of the management cluster.
Whenever a Machine leaves a cluster, it will join the management cluster again.