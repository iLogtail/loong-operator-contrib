English | [简体中文](README_zh.md)
# Kubernetes Operator for LoongCollector

Kubernetes Operator for LoongCollector is used to manage and deliver LoongCollector Pipelines with Config-Server integration

## Architecture

![architecture-design](docs/image/architecture-design.png)

## Features

- Support managing LoongCollector Pipeline configurations through Kubernetes CRD
- Automatically synchronize Pipeline configurations to LoongCollector Config-Server
- Support configuration validation and error handling
- Support configuration retry mechanism
- Support graceful deletion and resource cleanup
- Support configuring Config-Server address through ConfigMap

## Installation

### Prerequisites

- Kubernetes cluster 1.16+
- LoongCollector is deployed and running
  - Please refer to the [LoongCollector deployment documentation](https://ilogtail.gitbook.io/ilogtail-docs/installation/start-with-k8s) for deployment, or [quick deploy](config/samples/loongcollector.yaml) in the kubernetes cluster
- Config-Server is deployed and running

  - Please refer to the [Config-Server deployment documentation](https://github.com/iLogtail/ConfigServer) for deployment, or [quick deploy](config/samples/config-server/config-server.yaml) in the kubernetes cluster

### Quick Start

- Install Operator
```bash
kubectl apply -f https://github.com/infraflows/loongcollector-operator/blob/main/dist/install.yaml
```

- Deploy Config-Server (Optional):

```bash
kubectl apply -f https://github.com/infraflows/loongcollector-operator/blob/main/config/samples/infraflow_v1alpha1_pipeline.yaml
```

- Deploy LoongCollector (Optional):

```bash
kubectl apply -f https://github.com/infraflows/loongcollector-operator/blob/main/config/samples/loongcollector.yaml
```

## Usage

- Create Pipeline

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: infraflow.co/v1alpha1
kind: Pipeline
metadata:
  name: sample-pipeline
spec:
  name: sample-pipeline
  content: |
    tags:
      - default
    inputs:
      - type: file
        path: /var/log/containers/*.log
    processors:
      - type: json
        fields:
          - message
    outputs:
      - type: stdout
EOF

kubectl apply -f pipeline.yaml
```
Or use the following command:
```bash
kubectl apply -f https://github.com/infraflows/loongcollector-operator/blob/main/config/samples/infraflow_v1alpha1_pipeline.yaml
```

- Create AgentGroup

```bash
kubectl apply -f https://github.com/infraflows/loongcollector-operator/blob/main/config/samples/agentgroup.yaml
```

### Configuration Description

#### Pipeline CRD

Pipeline CRD fields description:

- `spec.name`: Pipeline name
- `spec.content`: Pipeline configuration (YAML format)

For more information on the Pipeline CRD fields, please refer to [Pipeline CRD documentation](docs/pipeline-fields.md)
#### Config-Server Configuration

The default Config-Server service address is `http://config-server:9090`, and the Config-Server address can also be configured through ConfigMap:

```yaml
apiVersion: v1alpha1
kind: ConfigMap
metadata:
  name: config-server-config
  namespace: loongcollector-system
  labels:
    app: config-server
data:
  configServerURL: "http://config-server:9090"
```
> Tips:
>- The priority of operator getting Config-Server is **default address `http://config-server:9090`** -> **ConfigMap**，the way to get ConfigMap is through label, the value is `app: config-server`, currently not supported to modify
>- If the Config-Server address changes, you need to manually update the ConfigMap and restart the operator
>   - `kubectl rollout restart deployment -n loongcollector-system loongcollector-operator` restart operator

## Development

### Local Development

1. Install dependencies:

```bash
go mod tidy
```

2. Run tests:

```bash
make test
```

3. Build image:

```bash
make docker-build
```

4. Generate installation file:

```bash
make build-installer
```

## License

[Apache License 2.0](LICENSE)