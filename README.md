# k8s-loki-duration-tracker

This Go program checks the log availability duration for Kubernetes Pods in a specified namespace prefix. It uses the Loki logging system to retrieve the logs and determines how long it takes for the logs to become available after the Pod starts.

This program uses the `query_range` of the Loki HTTP API.

https://grafana.com/docs/loki/latest/reference/api/#query-logs-within-a-range-of-time

## Futures

This program needs to be run at the same time as [zinrai/k8s-pod-log-generator](https://github.com/zinrai/k8s-loki-logline-verifier), as it calculates the difference between the start time of the k8s Pod and the difference that could be logged from Loki.

If you know of a better way to do this, please let me know.

It would be nice if we could make it so that it can calculate the time it takes to get the logs from Loki even after the [zinrai/k8s-pod-log-generator](https://github.com/zinrai/k8s-loki-logline-verifier) has been executed.

## Motivation

I wanted to measure the time it took for the logs to become searchable at Loki.

## Tested Version

- `Loki`: 2.9.5
    - https://grafana.com/docs/loki/latest/setup/install/helm/install-scalable/
- `Promtail`: 2.9.3
    - https://grafana.com/docs/loki/latest/send-data/promtail/installation/#install-using-helm

## Requirements

- Access to a Grafana Loki instance with `auth_enabled: true`
    - https://grafana.com/docs/loki/latest/configure/#supported-contents-and-default-values-of-lokiyaml
- Access to Loki search endpoints deployed on k8s is required.
- k8s Namespace is set as the unit of tenant id in Loki.

Example of access to a Loki search endpoint using port forwarding:

```
$ kubectl get service -n loki
NAME                        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)             AGE
loki-backend                ClusterIP   10.24.124.202   <none>        3100/TCP,9095/TCP   20d
loki-backend-headless       ClusterIP   None            <none>        3100/TCP,9095/TCP   20d
loki-gateway                ClusterIP   10.24.120.118   <none>        80/TCP              20d
loki-memberlist             ClusterIP   None            <none>        7946/TCP            20d
loki-read                   ClusterIP   10.24.123.180   <none>        3100/TCP,9095/TCP   20d
loki-read-headless          ClusterIP   None            <none>        3100/TCP,9095/TCP   20d
loki-write                  ClusterIP   10.24.120.24    <none>        3100/TCP,9095/TCP   20d
loki-write-headless         ClusterIP   None            <none>        3100/TCP,9095/TCP   20d
query-scheduler-discovery   ClusterIP   None            <none>        3100/TCP,9095/TCP   20d
$
```
```
$ kubectl port-forward svc/loki-gateway 8080:80 -n loki
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
Handling connection for 8080
Handling connection for 8080
```

Example values.yaml for Promtail using Helm when logging from Cloud Pub/Sub:

```
daemonset:
  enabled: false
deployment:
  enabled: true
serviceAccount:
  create: false
  name: ksa-cloudpubsub
configmap:
  enabled: true
config:
  clients:
    - url: http://loki-gateway.loki.svc.cluster.local/loki/api/v1/push
      tenant_id: default
  snippets:
    scrapeConfigs: |
      - job_name: gcplog
        pipeline_stages:
          - tenant:
              label: "namespace"
        gcplog:
          subscription_type: "pull"
          project_id: "project-id"
          subscription: "subscription-id"
          use_incoming_timestamp: false
          use_full_line: false
          labels:
            job: "gcplog"
        relabel_configs:
          - source_labels: ['__gcp_resource_labels_namespace_name']
            target_label: 'namespace'
          - source_labels: ['__gcp_resource_labels_pod_name']
            target_label: 'pod_name'
```

## Configuration

The program reads configurations from a YAML file named `config.yaml`. The following configuration options are available:

- `kubeconfig_path`: (Optional) Path to the Kubernetes cluster configuration file. If not provided, the default path will be used.
- `namespace_prefix`: (Optional) Prefix for the namespaces created by the tool. Defaults to logger-ns.
- `loki_address`: URL of the Loki server.

## Usage

```bash
$ cat << EOF > config.yaml
loki_address: "http://localhost:8080"
EOF
```

```bash
$ go run main.go
2024/04/12 17:00:04 Pod logger-pod-16 in namespace logger-ns-1 started at 2024-04-12T16:29:18+09:00. Logs were available after 30m46.743518849s.
2024/04/12 17:00:04 Pod logger-pod-36 in namespace logger-ns-1 started at 2024-04-12T16:32:21+09:00. Logs were available after 27m43.748676849s.
2024/04/12 17:00:04 Pod logger-pod-42 in namespace logger-ns-10 started at 2024-04-12T16:33:31+09:00. Logs were available after 26m33.816167474s.
2024/04/12 17:00:04 Pod logger-pod-52 in namespace logger-ns-10 started at 2024-04-12T16:35:52+09:00. Logs were available after 24m12.819209516s.
2024/04/12 17:00:04 Pod logger-pod-46 in namespace logger-ns-1 started at 2024-04-12T16:34:26+09:00. Logs were available after 25m38.854679933s.
2024/04/12 17:00:04 Pod logger-pod-60 in namespace logger-ns-1 started at 2024-04-12T16:38:05+09:00. Logs were available after 21m59.863043224s.
2024/04/12 17:00:04 Pod logger-pod-59 in namespace logger-ns-10 started at 2024-04-12T16:38:03+09:00. Logs were available after 22m1.889092308s.
2024/04/12 17:00:04 Pod logger-pod-39 in namespace logger-ns-1 started at 2024-04-12T16:32:58+09:00. Logs were available after 27m6.889749974s.
2024/04/12 17:00:04 Pod logger-pod-18 in namespace logger-ns-1 started at 2024-04-12T16:29:22+09:00. Logs were available after 30m42.891915558s.
2024/04/12 17:00:04 Pod logger-pod-33 in namespace logger-ns-10 started at 2024-04-12T16:31:22+09:00. Logs were available after 28m42.893805391s.
2024/04/12 17:00:04 Pod logger-pod-40 in namespace logger-ns-10 started at 2024-04-12T16:33:06+09:00. Logs were available after 26m58.896818974s.
2024/04/12 17:00:04 Pod logger-pod-17 in namespace logger-ns-2 started at 2024-04-12T16:29:20+09:00. Logs were available after 30m44.900347641s.
2024/04/12 17:00:04 Pod logger-pod-31 in namespace logger-ns-2 started at 2024-04-12T16:30:57+09:00. Logs were available after 29m7.905898558s.
2024/04/12 17:00:04 Pod logger-pod-41 in namespace logger-ns-2 started at 2024-04-12T16:33:18+09:00. Logs were available after 26m46.932350516s.
2024/04/12 17:00:04 Pod logger-pod-43 in namespace logger-ns-2 started at 2024-04-12T16:33:33+09:00. Logs were available after 26m31.933692099s.
...
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
