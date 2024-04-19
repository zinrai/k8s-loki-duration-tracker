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
2024/04/19 09:14:03 Error getting logs for pod logger-pod-3 in namespace logger-ns-1: no logs found for pod logger-pod-3
2024/04/19 09:14:04 Error getting logs for pod logger-pod-5 in namespace logger-ns-1: no logs found for pod logger-pod-5
2024/04/19 09:14:04 First log line for pod logger-pod-1 in namespace logger-ns-10: (Time difference: 7.481053523s)
2024/04/19 09:14:04 Error getting logs for pod logger-pod-6 in namespace logger-ns-4: no logs found for pod logger-pod-6
2024/04/19 09:14:04 Error getting logs for pod logger-pod-9 in namespace logger-ns-4: no logs found for pod logger-pod-9
2024/04/19 09:14:04 First log line for pod logger-pod-2 in namespace logger-ns-8: (Time difference: 6.924065065s)
2024/04/19 09:14:05 First log line for pod logger-pod-4 in namespace logger-ns-9: (Time difference: 6.288570148s)
2024/04/19 09:14:05 Error getting logs for pod logger-pod-7 in namespace logger-ns-9: no logs found for pod logger-pod-7
2024/04/19 09:14:05 First log line for pod logger-pod-3 in namespace logger-ns-1: (Time difference: 7.673499773s)
2024/04/19 09:14:05 Error getting logs for pod logger-pod-5 in namespace logger-ns-1: no logs found for pod logger-pod-5
2024/04/19 09:14:05 Error getting logs for pod logger-pod-8 in namespace logger-ns-1: no logs found for pod logger-pod-8
2024/04/19 09:14:05 Error getting logs for pod logger-pod-6 in namespace logger-ns-4: no logs found for pod logger-pod-6
2024/04/19 09:14:05 Error getting logs for pod logger-pod-9 in namespace logger-ns-4: no logs found for pod logger-pod-9
2024/04/19 09:14:06 Error getting logs for pod logger-pod-7 in namespace logger-ns-9: no logs found for pod logger-pod-7
2024/04/19 09:14:07 Error getting logs for pod logger-pod-5 in namespace logger-ns-1: no logs found for pod logger-pod-5
2024/04/19 09:14:07 First log line for pod logger-pod-8 in namespace logger-ns-1: (Time difference: 4.905360316s)
2024/04/19 09:14:07 Error getting logs for pod logger-pod-6 in namespace logger-ns-4: no logs found for pod logger-pod-6
...
2024/04/19 10:13:22 Error getting logs for pod logger-pod-3335 in namespace logger-ns-7: no logs found for pod logger-pod-3335
2024/04/19 10:13:22 Error getting logs for pod logger-pod-3337 in namespace logger-ns-7: no logs found for pod logger-pod-3337
2024/04/19 10:13:24 Error getting logs for pod logger-pod-3338 in namespace logger-ns-10: no logs found for pod logger-pod-3338
2024/04/19 10:13:24 Error getting logs for pod logger-pod-3340 in namespace logger-ns-10: no logs found for pod logger-pod-3340
2024/04/19 10:13:59 First log line for pod logger-pod-3368 in namespace logger-ns-4: (Time difference: 8.503452195s)
2024/04/19 10:13:59 Error getting logs for pod logger-pod-3372 in namespace logger-ns-4: no logs found for pod logger-pod-3372
2024/04/19 10:13:59 First log line for pod logger-pod-3371 in namespace logger-ns-6: (Time difference: 4.560198237s)
2024/04/19 10:13:59 First log line for pod logger-pod-3370 in namespace logger-ns-8: (Time difference: 6.590553446s)
2024/04/19 10:14:01 First log line for pod logger-pod-3369 in namespace logger-ns-1: (Time difference: 9.670723447s)
2024/04/19 10:14:01 Error getting logs for pod logger-pod-3372 in namespace logger-ns-4: no logs found for pod logger-pod-3372
2024/04/19 10:14:03 First log line for pod logger-pod-3372 in namespace logger-ns-4: (Time difference: 8.869745823s)
^CReceived signal: interrupt
Namespace: logger-ns-10, Pod: logger-pod-1, Time Difference: 7.481053523s
Namespace: logger-ns-8, Pod: logger-pod-2, Time Difference: 6.924065065s
Namespace: logger-ns-9, Pod: logger-pod-4, Time Difference: 6.288570148s
Namespace: logger-ns-1, Pod: logger-pod-3, Time Difference: 7.673499773s
Namespace: logger-ns-1, Pod: logger-pod-8, Time Difference: 4.905360316s
Namespace: logger-ns-9, Pod: logger-pod-7, Time Difference: 5.112803733s
Namespace: logger-ns-1, Pod: logger-pod-5, Time Difference: 9.102882317s
Namespace: logger-ns-4, Pod: logger-pod-6, Time Difference: 9.156968901s
Namespace: logger-ns-2, Pod: logger-pod-501, Time Difference: 4.554813477s
Namespace: logger-ns-1, Pod: logger-pod-649, Time Difference: 4.542230846s
Namespace: logger-ns-5, Pod: logger-pod-646, Time Difference: 9.979003179s
...
Namespace: logger-ns-8, Pod: logger-pod-3370, Time Difference: 6.590553446s
Namespace: logger-ns-1, Pod: logger-pod-3369, Time Difference: 9.670723447s
Namespace: logger-ns-4, Pod: logger-pod-3372, Time Difference: 8.869745823s

Max Time Difference: 12.22648255s
Min Time Difference: 2.538977783s
Mean Time Difference: 6.791767569s
$
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
