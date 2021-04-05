# Developing the Agent Operator

Create a k3d cluster (depending on k3d v4.x):

```
k3d cluster create agent-operator \
  --port 30080:80@loadbalancer \
  --api-port 50043 \
  --kubeconfig-update-default=true \
  --kubeconfig-switch-context=true \
  --wait
```

Run the operator:

```
go run ./cmd/agent-operator -server.http-listen-port=8080 -apiserver.use-kubeconfig
```
