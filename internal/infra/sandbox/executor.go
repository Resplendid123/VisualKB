package sandbox

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Thin shell; controller reconciles all.
type ClientGoExecutor struct {
	clientset *kubernetes.Clientset
}

func NewClientGoExecutor() (*ClientGoExecutor, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig failed: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("new clientset failed: %w", err)
	}
	return &ClientGoExecutor{
		clientset: clientset,
	}, nil
}

// Reserved for future debug tooling.
func (c *ClientGoExecutor) Clientset() *kubernetes.Clientset { return c.clientset }

// Merge stdout/stderr with newline.
func mergeExecOutput(stdout, stderr string) string {
	switch {
	case stdout == "":
		return stderr
	case stderr == "":
		return stdout
	default:
		return stdout + "\n" + stderr
	}
}
