package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ANSI Escape Codes
const (
	ColorReset  = "\u001b[0m"
	ColorRed    = "\u001b[1;31m"
	ColorYellow = "\u001b[1;33m"
	ColorGreen  = "\u001b[1;32m"
)

// Custom flag type to support repeated regex flags
type regexSlice []*regexp.Regexp

func (r *regexSlice) String() string {
	var strs []string
	for _, re := range *r {
		strs = append(strs, re.String())
	}
	return strings.Join(strs, ", ")
}

func (r *regexSlice) Set(value string) error {
	re, err := regexp.Compile(value)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", value, err)
	}
	*r = append(*r, re)
	return nil
}

// EnvoyLog represents the Envoy JSON access log schema
type EnvoyLog struct {
	StartTime              string  `json:"start_time"`
	Method                 *string `json:"method"`
	RequestedServerName    *string `json:"requested_server_name"`
	XEnvoyOriginPath       *string `json:"x-envoy-origin-path"`
	ResponseCode           int     `json:"response_code"`
	Duration               int64   `json:"duration"`
	RealRemoteAddress      *string `json:"real_remote_address"`
	UserAgent              *string `json:"user-agent"`
	RouteName              *string `json:"route_name"`
	ResponseFlags          *string `json:"response_flags"`
	ResponseFlagsLong      *string `json:"response_flags_long"`
	ResponseCodeDetails    *string `json:"response_code_details"`
	DownstreamLocalAddress *string `json:"downstream_local_address"`
}

func strOr(ptr *string, fallback string) string {
	if ptr == nil || *ptr == "" {
		return fallback
	}
	return *ptr
}

func main() {
	var includes regexSlice
	var excludes regexSlice

	// Repeated regex flags
	flag.Var(&includes, "include", "Regex pattern to include (can be repeated)")
	flag.Var(&includes, "i", "Regex pattern to include (shorthand)")
	flag.Var(&excludes, "exclude", "Regex pattern to exclude (can be repeated)")
	flag.Var(&excludes, "e", "Regex pattern to exclude (shorthand)")

	// K8s & general flags
	namespace := flag.String("namespace", "envoy-gateway-system", "Kubernetes namespace")
	labelSelector := flag.String("l", "gateway.envoyproxy.io/owning-gateway-name=main", "Pod label selector")
	containerName := flag.String("container", "envoy", "Container name")
	kubeconfig := flag.String("kubeconfig", "", "Optional path to explicit kubeconfig file")
	tailLines := flag.Int64("tail", 1, "Lines of recent log history to show")
	flag.Parse()

	// Fallback to EGLOGS_GREP env var if no -e/-exclude passed
	if len(excludes) == 0 {
		if envGrep := os.Getenv("EGLOGS_GREP"); envGrep != "" {
			pattern := strings.TrimPrefix(envGrep, "-v -e ")
			if re, err := regexp.Compile(pattern); err == nil {
				excludes = append(excludes, re)
			}
		}
	}

	// Standard Kubernetes client configuration loading:
	// Automatically respects $KUBECONFIG environment variable, fallback to ~/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if *kubeconfig != "" {
		loadingRules.ExplicitPath = *kubeconfig
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating clientset: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods(*namespace).List(ctx, metav1.ListOptions{
		LabelSelector: *labelSelector,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing pods: %v\n", err)
		os.Exit(1)
	}

	if len(pods.Items) == 0 {
		fmt.Printf("No pods found matching selector %q in namespace %q\n", *labelSelector, *namespace)
		return
	}

	var wg sync.WaitGroup
	var outMutex sync.Mutex

	for _, pod := range pods.Items {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()

			req := clientset.CoreV1().Pods(*namespace).GetLogs(podName, &corev1.PodLogOptions{
				Container: *containerName,
				Follow:    true,
				TailLines: tailLines,
			})

			stream, err := req.Stream(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error streaming logs for pod %s: %v\n", podName, err)
				return
			}
			defer stream.Close()

			scanner := bufio.NewScanner(stream)
			for scanner.Scan() {
				line := scanner.Text()

				if !shouldProcessLine(line, includes, excludes) {
					continue
				}

				formatted := formatLogLine(line)
				if formatted != "" {
					outMutex.Lock()
					fmt.Println(formatted)
					outMutex.Unlock()
				}
			}
		}(pod.Name)
	}

	wg.Wait()
}

func shouldProcessLine(line string, includes, excludes regexSlice) bool {
	if len(includes) > 0 {
		matched := false
		for _, re := range includes {
			if re.MatchString(line) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, re := range excludes {
		if re.MatchString(line) {
			return false
		}
	}

	return true
}

func formatLogLine(raw string) string {
	var log EnvoyLog
	if err := json.Unmarshal([]byte(raw), &log); err != nil {
		return raw
	}

	// 1. Condition: route_name == null
	if log.RouteName == nil {
		flags := strOr(log.ResponseFlags, "-")
		flagsLong := strOr(log.ResponseFlagsLong, "-")
		details := strOr(log.ResponseCodeDetails, "-")
		remoteAddr := strOr(log.RealRemoteAddress, "-")
		localAddr := strOr(log.DownstreamLocalAddress, "-")

		return fmt.Sprintf(
			"[%s] %s%s (%s)%s (%s%s%s) ➜ %s%d%s (%dms) | %s → %s",
			log.StartTime,
			ColorRed, flags, flagsLong, ColorReset,
			ColorRed, details, ColorReset,
			ColorRed, log.ResponseCode, ColorReset,
			log.Duration,
			remoteAddr, localAddr,
		)
	}

	// 2. Standard Request
	var statusColor string
	if log.ResponseCode >= 500 {
		statusColor = ColorRed
	} else if log.ResponseCode >= 400 {
		statusColor = ColorYellow
	} else {
		statusColor = ColorGreen
	}

	method := strOr(log.Method, "-")
	sni := strOr(log.RequestedServerName, "")
	path := strOr(log.XEnvoyOriginPath, "")
	remoteAddr := strOr(log.RealRemoteAddress, "-")
	userAgent := strOr(log.UserAgent, "-")

	return fmt.Sprintf(
		"[%s] %s %s%s ➜ %s%d%s (%dms) | %s %s",
		log.StartTime,
		method,
		sni, path,
		statusColor, log.ResponseCode, ColorReset,
		log.Duration,
		remoteAddr, userAgent,
	)
}
