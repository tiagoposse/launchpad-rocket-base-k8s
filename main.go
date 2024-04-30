package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podName := os.Getenv("MY_POD_NAME")
	namespace := os.Getenv("MY_POD_NAMESPACE")

	// Define a function to check if Container A in the Pod has completed
	checkContainerACompletion := func() error {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err != nil {
			return err
		}

		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == "main" {
				if status.State.Terminated != nil {
					fmt.Println("main has completed.")
					return nil
				}
				fmt.Println("Container A is still running...")
				return fmt.Errorf("container A is not yet completed")
			}
		}

		return fmt.Errorf("container A not found in the Pod status")
	}

	// Poll for Container A completion status with retries
	err = retry.RetryOnConflict(retry.DefaultRetry, checkContainerACompletion)
	if err != nil {
		fmt.Println("Error:", err)
		// Perform actions in case of an error (e.g., handle timeout)
		return
	}

	// Container A has completed, now perform API call or desired action
	err = finishLaunch(context.TODO())
	if err != nil {
		fmt.Println("Error calling API:", err)
		return
	}
	fmt.Println("API called successfully")
}

func finishLaunch(ctx context.Context) error {
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// Read the service account token
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token: %v", err)
	}

	url := filepath.Join(os.Getenv("LAUNCHPAD_API_URL"), "api", "mission", "launched", string(token))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating http req: %w", err)
	}

	httpRes, err := (&http.Client{}).Do(httpReq)
	if err != nil {
		return fmt.Errorf("executing http req: %w", err)
	}
	if httpRes.StatusCode != 200 {
		return fmt.Errorf("request failed with status: %w", err)
	}

	httpRes.Body.Close()
	return nil
}
