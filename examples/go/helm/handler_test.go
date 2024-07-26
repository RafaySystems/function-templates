package function_test

import (
	"context"
	function "handler/helm-function"
	"log/slog"
	"os"
	"testing"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"github.com/testcontainers/testcontainers-go/modules/k3s"
)

func TestHelmHandler(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	k8s, err := k3s.Run(context.TODO(), "docker.io/rancher/k3s:v1.27.1-k3s1")
	if err != nil {
		t.Fatalf("failed to run k3s container: %v", err)
	}

	defer k8s.Terminate(context.Background())

	kubeconfig, err := k8s.GetKubeConfig(context.TODO())
	if err != nil {
		t.Fatalf("failed to get kubeconfig: %v", err)
	}

	req := sdk.Request{
		"action":        "deploy",
		"namespace":     "default",
		"release":       "my-release",
		"repo_url":      "oci://registry-1.docker.io/bitnamicharts/redis",
		"chart_version": "19.6.1",
		"kubeconfig":    kubeconfig,
		"helm_values":   map[string]interface{}{},
	}

	resp, err := function.Handle(context.TODO(), logger, req)
	if err != nil {
		t.Fatalf("failed to install helm chart: %v", err)
	}
	t.Log("install response: ", resp)

	resp, err = function.Handle(context.TODO(), logger, req)
	if err != nil {
		t.Fatalf("failed to upgrade helm chart: %v", err)
	}
	t.Log("upgrade response: ", resp)

	req["action"] = "destroy"
	resp, err = function.Handle(context.TODO(), logger, req)
	if err != nil {
		t.Fatalf("failed to deploy helm chart: %v", err)
	}
	t.Log("destroy response: ", resp)
}
