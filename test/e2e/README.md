# E2E tests for the operator

E2E tests run on a Kind cluster that is set up by the make target. In the case of an error, the cluster stays intact, and can be removed by calling: `kind delete clusters intel-gpu-base-operator-test-e2e`

The node where the e2e tests are run has to have one or more Intel GPUs with suitable KMD installed. While the operator prioritizes the newer `xe` KMD, it also supports the `i915`.

## Dependencies

Runtime dependencies:

* make - to trigger e2e run
* go - to compile and run the e2e tests
* kubectl - to deploy the operator and overall access to the cluster
* helm - to deploy the operator
* kind - to setup a temporary cluster
* jq - to parse json data in tests

Helm and Kind can be installed using `make`:
```
mkdir -p $(pwd)/bin
export PATH=$PATH:$(pwd)/bin
make helm
make kind
```

### Collateral images

Test run automatically pre-pulls the images used in the run to prevent timeouts during the test execution.
```
  STEP: loading the manager(Operator) image on Kind @ 12/03/25 10:33:33.238
  STEP: loading collateral images on Kind @ 12/03/25 10:33:33.349
  STEP: loading image intel/intel-gpu-plugin:0.34.0 to kind cluster @ 12/03/25 10:33:33.35
  STEP: loading image intel/intel-gpu-levelzero:0.34.0 to kind cluster @ 12/03/25 10:33:35.087
  STEP: loading image intel/xpumanager:v1.2.27 to kind cluster @ 12/03/25 10:33:36.941
  STEP: loading image ghcr.io/intel/intel-resource-drivers-for-kubernetes/intel-gpu-resource-driver:v0.9.0 to kind cluster @ 12/03/25 10:33:38.787
```

## How to run

Build the latest operator:

```
make docker-build
```
> Test run automatically imports the operator image to Kind from docker.


And run the tests for nodes with newer `xe` KMD (i915 tests are excluded):

```
make test-e2e
```

For nodes with older `i915` KMD (xe tests are excluded):

```
make test-e2e-i915
```

The separation of `xe` and `i915` is due to how the Intel GPU plugin registers two different monitoring resources. The XPU Manager has to have a Daemonset targeted for either resource.
