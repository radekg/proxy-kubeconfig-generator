It generates a Kubeconfig Secret from a Service Account and a Kubernetes API server proxy.

## Synopsis

```
proxy-kubeconfig-generator \
    --serviceaccount <Service Account name> \
    --server <server url> \
    --server-tls-secret-name <server TLS Secret name> \
    [options]
```

## Options

```
  -serviceaccount string
    	The name of the service account for which to create the kubeconfig
  -namespace string
    	(optional) The namespace of the service account and where the kubeconfig secret will be created. (default "default")
  -server string
    	The server url of the kubeconfig where API requests will be sent
  -server-tls-secret-name string
    	The server TLS secret name
  -server-tls-secret-ca-key string
    	(optional) The CA key in the server TLS secret. (default "ca")
  -server-tls-secret-namespace string
    	(optional) The namespace of the server TLS secret. (default "default")
  -kubeconfig-secret-key string
    	(optional) The key of the kubeconfig in the secret that will be created (default "kubeconfig")
```


## Quick start

### Build and pull the OCI image

```
pack build generator --path .
kind create cluster
kind load docker-image generator:latest
```


### Deploy the generator

```
kubectl apply -k ./deploy/generator
```

### Give it a try

**Create a cluster**
```
kind create cluster
```

**Add Clastix Helm Repository**
```
helm repo add clastix https://clastix.github.io/charts
```

**Deploy Capsule**
```
helm upgrade --install -n default clastix/capsule
```

**Deploy Capsule Proxy**
```
helm upgrade --install -n default clastix/capsule-proxy
```

**Deploy a Tenant**
```
kubectl apply -f ./deploy/tenant/dev-team
```

**Deploy the generator Job**
```
pack build generator --path .
kind load docker-image generator:latest
kubectl apply -k ./deploy/generator
```

**Check the result**
```
$ kubectl get secret -n dev-team gitops-reconciler-kubeconfig
NAME                          TYPE    DATA  AGE
gitops-reconciler-kubeconfig  Opaque  1     1s
```