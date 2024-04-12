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
	"sync"
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
		Stats struct{} `json:"stats"`
	} `json:"data"`
}

type PodInfo struct {
	Namespace string
	PodName   string
	StartTime time.Time
}

func getLogAvailabilityDuration(podInfo PodInfo, lokiAddress string) (time.Duration, error) {
	startedAt := podInfo.StartTime.UnixNano()
	end := time.Now().UnixNano()

	query := fmt.Sprintf(`count_over_time({pod_name="%s"}[1h])`, podInfo.PodName)

	u, err := url.Parse(fmt.Sprintf("%s/loki/api/v1/query_range", lokiAddress))
	if err != nil {
		return 0, err
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(startedAt, 10))
	params.Set("end", strconv.FormatInt(end, 10))
	params.Set("step", "1h")
	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Scope-OrgID", podInfo.Namespace)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Loki query failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var lokiResp LokiQueryRangeResponse
	err = json.Unmarshal(body, &lokiResp)
	if err != nil {
		return 0, err
	}

	if len(lokiResp.Data.Result) > 0 {
		return time.Since(podInfo.StartTime), nil
	}

	return 0, fmt.Errorf("no logs found for pod %s", podInfo.PodName)
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

	var podQueue = make(chan PodInfo, 10000)
	var wg sync.WaitGroup
	var loggedPods = make(map[string]bool)
	var loggedPodsMutex sync.Mutex

	// Update the pod queue
	go func() {
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
					if pod.Status.StartTime != nil {
						podInfo := PodInfo{
							Namespace: namespace.Name,
							PodName:   pod.Name,
							StartTime: pod.Status.StartTime.Time,
						}
						podQueue <- podInfo
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Process the pod queue
	for i := 0; i < 10; i++ { // Start 10 worker goroutines
		wg.Add(1)
		go func() {
			defer wg.Done()
			for podInfo := range podQueue {
				loggedPodsMutex.Lock()
				if _, exists := loggedPods[podInfo.Namespace+"/"+podInfo.PodName]; exists {
					loggedPodsMutex.Unlock()
					continue
				}
				loggedPodsMutex.Unlock()

				logAvailabilityDuration, err := getLogAvailabilityDuration(podInfo, config.LokiAddress)
				if err != nil {
					continue
				}

				log.Printf("Pod %s in namespace %s started at %s. Logs were available after %s.", podInfo.PodName, podInfo.Namespace, podInfo.StartTime.Format(time.RFC3339), logAvailabilityDuration)

				loggedPodsMutex.Lock()
				loggedPods[podInfo.Namespace+"/"+podInfo.PodName] = true
				loggedPodsMutex.Unlock()
			}
		}()
	}

	wg.Wait()
}

func isTargetNamespace(namespaceName, namespacePrefix string) bool {
	return len(namespaceName) >= len(namespacePrefix) && namespaceName[:len(namespacePrefix)] == namespacePrefix
}
