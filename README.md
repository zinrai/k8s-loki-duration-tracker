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
2024/04/17 17:18:42 Error getting logs for pod logger-pod-1 in namespace logger-ns-1: no logs found for pod logger-pod-1
2024/04/17 17:18:42 Error getting logs for pod logger-pod-2 in namespace logger-ns-5: no logs found for pod logger-pod-2
2024/04/17 17:18:44 Error getting logs for pod logger-pod-1 in namespace logger-ns-1: no logs found for pod logger-pod-1
2024/04/17 17:18:44 Error getting logs for pod logger-pod-2 in namespace logger-ns-5: no logs found for pod logger-pod-2
2024/04/17 17:18:46 First log line for pod logger-pod-1 in namespace logger-ns-1: (Time difference: 7.783215399s)
2024/04/17 17:18:46 Error getting logs for pod logger-pod-3 in namespace logger-ns-3: no logs found for pod logger-pod-3
2024/04/17 17:18:46 First log line for pod logger-pod-2 in namespace logger-ns-5: (Time difference: 5.844791982s)
2024/04/17 17:18:49 Error getting logs for pod logger-pod-4 in namespace logger-ns-1: no logs found for pod logger-pod-4
2024/04/17 17:18:49 Error getting logs for pod logger-pod-3 in namespace logger-ns-3: no logs found for pod logger-pod-3
2024/04/17 17:18:51 First log line for pod logger-pod-4 in namespace logger-ns-1: (Time difference: 5.202028568s)
2024/04/17 17:18:51 Error getting logs for pod logger-pod-5 in namespace logger-ns-10: no logs found for pod logger-pod-5
2024/04/17 17:18:51 First log line for pod logger-pod-3 in namespace logger-ns-3: (Time difference: 7.319506151s)
2024/04/17 17:18:53 Error getting logs for pod logger-pod-5 in namespace logger-ns-10: no logs found for pod logger-pod-5
2024/04/17 17:18:53 Error getting logs for pod logger-pod-6 in namespace logger-ns-3: no logs found for pod logger-pod-6
2024/04/17 17:18:53 Error getting logs for pod logger-pod-7 in namespace logger-ns-9: no logs found for pod logger-pod-7
2024/04/17 17:18:55 Error getting logs for pod logger-pod-5 in namespace logger-ns-10: no logs found for pod logger-pod-5
2024/04/17 17:18:55 Error getting logs for pod logger-pod-6 in namespace logger-ns-3: no logs found for pod logger-pod-6
...
^CReceived signal: interrupt
Namespace: logger-ns-1, Pod: logger-pod-1, Time Difference: 7.783215399s
Namespace: logger-ns-5, Pod: logger-pod-2, Time Difference: 5.844791982s
Namespace: logger-ns-1, Pod: logger-pod-4, Time Difference: 5.202028568s
Namespace: logger-ns-3, Pod: logger-pod-3, Time Difference: 7.319506151s
Namespace: logger-ns-10, Pod: logger-pod-5, Time Difference: 9.799009404s
Namespace: logger-ns-3, Pod: logger-pod-6, Time Difference: 7.872321404s
Namespace: logger-ns-9, Pod: logger-pod-7, Time Difference: 6.033944321s
Namespace: logger-ns-4, Pod: logger-pod-8, Time Difference: 7.181646406s
Namespace: logger-ns-7, Pod: logger-pod-9, Time Difference: 5.334858406s
Namespace: logger-ns-7, Pod: logger-pod-10, Time Difference: 7.63087245s
Namespace: logger-ns-3, Pod: logger-pod-11, Time Difference: 7.803359284s
Namespace: logger-ns-9, Pod: logger-pod-12, Time Difference: 10.192467786s
Namespace: logger-ns-7, Pod: logger-pod-13, Time Difference: 7.63884571s
Namespace: logger-ns-1, Pod: logger-pod-14, Time Difference: 9.410498756s
Namespace: logger-ns-5, Pod: logger-pod-15, Time Difference: 7.414764053s
Namespace: logger-ns-1, Pod: logger-pod-16, Time Difference: 9.79947093s
Namespace: logger-ns-8, Pod: logger-pod-17, Time Difference: 7.939192514s
Namespace: logger-ns-6, Pod: logger-pod-18, Time Difference: 10.352854891s
Namespace: logger-ns-10, Pod: logger-pod-19, Time Difference: 6.633163809s
Namespace: logger-ns-6, Pod: logger-pod-20, Time Difference: 4.696835143s
Namespace: logger-ns-1, Pod: logger-pod-22, Time Difference: 7.194588438s
Namespace: logger-ns-8, Pod: logger-pod-21, Time Difference: 9.311871854s
Namespace: logger-ns-2, Pod: logger-pod-23, Time Difference: 9.60719519s
Namespace: logger-ns-9, Pod: logger-pod-24, Time Difference: 7.00249403s
Namespace: logger-ns-6, Pod: logger-pod-25, Time Difference: 11.012602119s
Namespace: logger-ns-7, Pod: logger-pod-26, Time Difference: 11.358674375s
Namespace: logger-ns-8, Pod: logger-pod-27, Time Difference: 9.392473959s
Namespace: logger-ns-8, Pod: logger-pod-28, Time Difference: 7.6111735s
Namespace: logger-ns-3, Pod: logger-pod-29, Time Difference: 8.067295088s
Namespace: logger-ns-8, Pod: logger-pod-30, Time Difference: 7.065616926s
Namespace: logger-ns-8, Pod: logger-pod-31, Time Difference: 13.08654281s
Namespace: logger-ns-1, Pod: logger-pod-33, Time Difference: 11.559511481s
Namespace: logger-ns-10, Pod: logger-pod-32, Time Difference: 13.613584481s
Namespace: logger-ns-5, Pod: logger-pod-34, Time Difference: 9.789898356s
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
