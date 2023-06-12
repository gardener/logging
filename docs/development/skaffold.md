# Development scenarios for the logging stack

## Skaffold based development scenario for fluent-bit-vali-plugin

It is possible to construct a local development environment for the logging stack based on [skaffold](https://skaffold.dev). It builds upon the gardener `skaffold` pipeline and provides a hook for building the deploying the `fluent-bit-plugin` into the local development kind cluster.

![skaffold pipeline](images/skaffold.png)

To bring the development environment first we need to bring a local kind cluster with a default set of workloads required by gardener. The simplest way to achieve this is to fetch the gardener repository into the local project folder and leverage the already existing targets.

To fetch a gardener repository, in case it is not already fetched, invoke the following command.
```bash
hack/fetch-gardener.sh
```

It tries to get the version of which gardener release to fetch from the dependency defined in the project `go.mod` file.
Once the gardener repository is fetched, initiate the kind cluster with:
```bash
make -C gardener kind-up
```

In addition to the kind cluster itself, the `kind-up` target brings also: a docker registry, calico and a metrics-server.
Once the kind cluster is up and running we can start the `skaffold` local pipeline with the following command:
```bash
make skaffold-up
```

This uses the skaffold.yaml definition to bring `etcd`, `controlplane`,  `provider local` and `gardenlet` skaffold modules into the local kind cluster. The last `gardenlet` module contains the build target for generating the `fluent-bit-plugin` container image and it pushes it to the local registry already present in the local kind cluster. Once the skaffold deployment is triggered the `fluent-bit-plugin` is brought into the cluster with the gardenlet specific deployment model.

After the `skaffold-up` target is executed the skaffold pipeline may be started in the `dev` mode with
```bash
make skaffold-dev
```

That brings the ability to run the entire pipeline once the local code modifications are made and ready to be tested.
