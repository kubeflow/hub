# Set up local development with kubeflow

**Note: this should only be needed in edge cases in which we need to test a local integration with the kubeflow dashboard.**

## Prerequisites

- [Kubeflow repo](https://github.com/kubeflow/dashboard/tree/main/components/centraldashboard#development)
- [Hub repo](../README.md)

## Setup

### Kubeflow repo (under centraldashboard)

1. Change the [webpack config](https://github.com/kubeflow/dashboard/blob/main/components/centraldashboard/webpack.config.js#L186) proxies to allow Model Registry:

```javascript
        proxy: {
            ...
            '/model-registry': {
                target: 'http://localhost:9000',
                pathRewrite: {'^/model-registry': ''},
                changeOrigin: true,
                ws: true,
                secure: false,
            },
        },
```

2. Run the centraldashboard:

```shell
npm run dev
```

### Hub repo

1. Just run the repo in kubeflow dev mode

```shell
make dev-start-kubeflow
```

### Access the cluster

You need to have a kubeflow cluster up and running, to get the Hub working you'll need to port-forward these two services:

```shell
kubectl port-forward service/model-registry-service 8085:8080 -n <targeted-namespace-of-the-mr-service>
```

```shell
kubectl port-forward svc/profiles-kfam 8081:8081 -n kubeflow
```

After setting up port forwarding, you can access the UI by navigating to:

http://localhost:8080
