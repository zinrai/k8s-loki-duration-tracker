package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	KubeconfigPath  string `yaml:"kubeconfig_path"`
	NamespacePrefix string `yaml:"namespace_prefix"`
	LokiAddress     string `yaml:"loki_address"`
}

type LokiQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type PodInfo struct {
	Namespace string
	PodName   string
	StartTime time.Time
}

type JobQueue struct {
	podInfoQueue []PodInfo
	loggedPods   map[string]bool
}

type LoggedPodInfo struct {
	Namespace string
	PodName   string
	TimeDiff  time.Duration
}

func (jq *JobQueue) AddPodToQueue(podInfo PodInfo) {
	jq.podInfoQueue = append(jq.podInfoQueue, podInfo)
}

func (jq *JobQueue) GetPodFromQueue() (PodInfo, bool) {
	if len(jq.podInfoQueue) == 0 {
		return PodInfo{}, false
	}
	podInfo := jq.podInfoQueue[0]
	jq.podInfoQueue = jq.podInfoQueue[1:]
	return podInfo, true
}

func (jq *JobQueue) MarkPodAsLogged(podInfo PodInfo) {
	jq.loggedPods[fmt.Sprintf("%s/%s", podInfo.Namespace, podInfo.PodName)] = true
}

func (jq *JobQueue) IsPodLogged(podInfo PodInfo) bool {
	_, ok := jq.loggedPods[fmt.Sprintf("%s/%s", podInfo.Namespace, podInfo.PodName)]
	return ok
}

func getLokiLogs(podInfo PodInfo, lokiAddress string) (LoggedPodInfo, error) {
	startedAt := podInfo.StartTime.UnixNano()

	query := fmt.Sprintf(`{pod_name="%s"}`, podInfo.PodName)
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(startedAt, 10))
	params.Set("end", strconv.FormatInt(time.Now().UnixNano(), 10))

	lokiURL := fmt.Sprintf("%s/loki/api/v1/query_range", lokiAddress)
	req, err := http.NewRequest("GET", lokiURL, nil)
	if err != nil {
		return LoggedPodInfo{}, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("X-Scope-OrgID", podInfo.Namespace)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return LoggedPodInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LoggedPodInfo{}, fmt.Errorf("failed to get logs from Loki: %s", resp.Status)
	}

	var lokiResp LokiQueryRangeResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LoggedPodInfo{}, err
	}
	err = json.Unmarshal(body, &lokiResp)
	if err != nil {
		return LoggedPodInfo{}, err
	}

	if len(lokiResp.Data.Result) > 0 {
		now := time.Now()
		timeDiff := now.Sub(podInfo.StartTime)
		log.Printf("First log line for pod %s in namespace %s: (Time difference: %s)", podInfo.PodName, podInfo.Namespace, timeDiff)
		return LoggedPodInfo{
			Namespace: podInfo.Namespace,
			PodName:   podInfo.PodName,
			TimeDiff:  timeDiff,
		}, nil
	}

	return LoggedPodInfo{}, fmt.Errorf("no logs found for pod %s", podInfo.PodName)
}

func main() {
	configFile := "config.yaml"

	configFileData, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer configFileData.Close()

	var config Config
	decoder := yaml.NewDecoder(configFileData)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	if config.NamespacePrefix == "" {
		config.NamespacePrefix = "logger-ns"
	}

	if config.KubeconfigPath == "" {
		config.KubeconfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
	if err != nil {
		log.Fatalf("Error building kubeconfig from %s: %v", config.KubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	jobQueue := JobQueue{
		loggedPods: make(map[string]bool),
	}

	var loggedPods []LoggedPodInfo
	emptyQueueCount := 0
	for {
		namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to list namespaces: %v", err)
		}

		for _, namespace := range namespaces.Items {
			if !isTargetNamespace(namespace.Name, config.NamespacePrefix) {
				continue
			}

			pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Fatalf("Failed to list pods in namespace %s: %v", namespace.Name, err)
			}

			for _, pod := range pods.Items {
				if pod.Status.StartTime != nil && !jobQueue.IsPodLogged(PodInfo{
					Namespace: namespace.Name,
					PodName:   pod.Name,
					StartTime: pod.Status.StartTime.Time,
				}) {
					podInfo := PodInfo{
						Namespace: namespace.Name,
						PodName:   pod.Name,
						StartTime: pod.Status.StartTime.Time,
					}
					jobQueue.AddPodToQueue(podInfo)
				}
			}
		}

		for {
			podInfo, ok := jobQueue.GetPodFromQueue()
			if !ok {
				emptyQueueCount++
				if emptyQueueCount >= 3 {
					fmt.Println("Program is terminating as the job queue has been empty 3 times.")
					printLoggedPods(loggedPods)
					return
				}
				break
			}

			loggedPodInfo, err := getLokiLogs(podInfo, config.LokiAddress)
			if err != nil {
				log.Printf("Error getting logs for pod %s in namespace %s: %v", podInfo.PodName, podInfo.Namespace, err)
			} else {
				jobQueue.MarkPodAsLogged(podInfo)
				loggedPods = append(loggedPods, loggedPodInfo)
			}
		}
	}
}

func isTargetNamespace(namespaceName, namespacePrefix string) bool {
	return len(namespaceName) >= len(namespacePrefix) && namespaceName[:len(namespacePrefix)] == namespacePrefix
}

func printLoggedPods(loggedPods []LoggedPodInfo) {
	for _, podInfo := range loggedPods {
		fmt.Printf("Namespace: %s, Pod: %s, Time Difference: %s\n", podInfo.Namespace, podInfo.PodName, podInfo.TimeDiff)
	}
}
