# kubecluster
[![Build Status](https://github.com/chriskery/kubecluster/actions/workflows/test-go.yml/badge.svg?branch=master)](https://github.com/chriskery/kubecluster/actions/workflows/test-go.yaml?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/chriskery/kubecluster/badge.svg?branch=master)](https://coveralls.io/github/chriskery/kubecluster?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/chriskery/kubecluster)](https://goreportcard.com/report/github.com/chriskery/kubecluster)

### The kubecluster implements a mechanism that makes it easy to build Slurm/Torque clusters on Kubernetes.

## Features
Kubecluster uses Pods to simulate nodes in different clusters, currently supports the following cluster types :

- [Slurm](pkg/controller/slurm_schema)
- [Torque( PBS )](pkg/controller/torque_schema)
## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

## Installation

### Master Branch

```bash
kubectl apply -k "github.com/chriskery/kubecluster/manifests/default"
```

## Quick Start

Please refer to the [quick-start.md](docs/quick-start.md) and [Kubeflow Training User Guide](https://www.kubeflow.org/docs/guides/components/tftraining/) for more information.


### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


