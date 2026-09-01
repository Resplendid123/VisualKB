package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"learn/internal/domain"
	"learn/internal/domain/project"
)

// CRD GVR; controller owns lifecycle.
var AgentSandboxGVK = schema.GroupVersionResource{
	Group:    "sandbox.sandbox.io",
	Version:  "v1",
	Resource: "agentsandboxes",
}

// CR name for project (pod suffix).
func SandboxName(projectID string) string {
	return "sandbox-" + projectID
}

// CRD subset we set; no Build field.
type AgentSandboxSpec struct {
	TenantID      string                      `json:"tenantID"`
	ProjectID     string                      `json:"projectID"`
	Runtime       RuntimeSection              `json:"runtime"`
	PVC           PVCSection                  `json:"pvc"`
	Resources     corev1.ResourceRequirements `json:"resources"`
	IdleTTL       string                      `json:"idleTTL,omitempty"` // e.g. "30m"
	PublicBaseURL string                      `json:"publicBaseURL,omitempty"`
}

type RuntimeSection struct {
	Image string          `json:"image"`
	Cmd   []string        `json:"cmd,omitempty"`
	Env   []corev1.EnvVar `json:"env,omitempty"`
}

type PVCSection struct {
	StorageClassName string `json:"storageClassName,omitempty"`
	Size             string `json:"size,omitempty"`
}

// Owns CR + exec via dynamic client.
type ControllerExecutor struct {
	dynamic dynamic.Interface

	// Controller exec base URL.
	execEndpoint string
	httpClient   *http.Client
	authToken    string

	// Per-user ns; "fixed" mode for tests.
	namespaceMode  string
	fixedNamespace string
}

// Default in-cluster controller endpoint.
const defaultExecEndpoint = "http://controller-exec.agent-sandbox-system.svc.cluster.local:8082"

// Build from kubeconfig (in/out cluster).
func NewControllerExecutor() (*ControllerExecutor, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("new dynamic client: %w", err)
	}
	endpoint := os.Getenv("SANDBOX_CONTROLLER_EXEC_URL")
	if endpoint == "" {
		endpoint = defaultExecEndpoint
	}
	return &ControllerExecutor{
		dynamic:       dyn,
		execEndpoint:  endpoint,
		httpClient:    &http.Client{},
		authToken:     os.Getenv("SANDBOX_EXEC_TOKEN"),
		namespaceMode: "perUser",
	}, nil
}

// Bypass ns derivation (tests).
func (c *ControllerExecutor) SetFixedNamespace(ns string) {
	c.namespaceMode = "fixed"
	c.fixedNamespace = ns
}

func (c *ControllerExecutor) namespaceFor(userID int64) string {
	if c.namespaceMode == "fixed" {
		return c.fixedNamespace
	}
	return project.UserNamespace(userID)
}

// Create/update AgentSandbox CR (idempotent).
func (c *ControllerExecutor) ApplySandbox(ctx context.Context, userID int64, projectID string, raw any) error {
	spec, ok := raw.(AgentSandboxSpec)
	if !ok {
		if ptr, ok := raw.(*AgentSandboxSpec); ok && ptr != nil {
			spec = *ptr
		} else {
			return fmt.Errorf("ApplySandbox: spec must be AgentSandboxSpec, got %T", raw)
		}
	}
	if spec.TenantID == "" {
		spec.TenantID = "u-" + strconv.FormatInt(userID, 10)
	}
	if spec.ProjectID == "" {
		spec.ProjectID = projectID
	}
	if spec.PVC.Size == "" {
		spec.PVC.Size = "10Gi"
	}
	if spec.PVC.StorageClassName == "" {
		// Fallback SC must match controller csi-s3 driver.
		spec.PVC.StorageClassName = "s3-csi-minio"
	}

	ns := c.namespaceFor(userID)
	if err := c.ensureNamespaceHTTP(ctx, ns, spec.TenantID); err != nil {
		return fmt.Errorf("ensure namespace %s: %w", ns, err)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": AgentSandboxGVK.Group + "/" + AgentSandboxGVK.Version,
			"kind":       "AgentSandbox",
			"metadata": map[string]any{
				"name":      SandboxName(projectID),
				"namespace": c.namespaceFor(userID),
				"labels": map[string]any{
					"agentsandbox.sandbox.io/tenant":  spec.TenantID,
					"agentsandbox.sandbox.io/project": spec.ProjectID,
					"app.kubernetes.io/managed-by":    "visualkb",
				},
			},
			"spec": specToMap(spec),
		},
	}

	res := c.dynamic.Resource(AgentSandboxGVK).Namespace(obj.GetNamespace())
	existing, err := res.Get(ctx, obj.GetName(), metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		// Fresh CR; create path next.
	case err != nil:
		return fmt.Errorf("get agentsandbox: %w", err)
	default:
		existing.Object["spec"] = obj.Object["spec"]
		obj = existing
	}
	if _, err := res.Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsInvalid(err) || apierrors.IsNotFound(err) {
			_, createErr := res.Create(ctx, obj, metav1.CreateOptions{})
			return createErr
		}
		return fmt.Errorf("apply agentsandbox: %w", err)
	}
	return nil
}

// EnsureNamespace asks controller to create ns.
func (c *ControllerExecutor) ensureNamespaceHTTP(ctx context.Context, ns, tenantID string) error {
	url := strings.TrimRight(c.execEndpoint, "/") + "/v1/namespaces/" + ns + "/ensure"
	body, err := json.Marshal(struct {
		TenantID string `json:"tenantID,omitempty"`
	}{TenantID: tenantID})
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("X-Sandbox-Token", c.authToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ensure-ns http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ensure-ns http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// JSON round-trip spec to map.
func specToMap(spec AgentSandboxSpec) map[string]any {
	buf, err := json.Marshal(spec)
	if err != nil {
		panic(fmt.Sprintf("marshal AgentSandboxSpec: %v", err))
	}
	out := map[string]any{}
	if err := json.Unmarshal(buf, &out); err != nil {
		panic(fmt.Sprintf("unmarshal AgentSandboxSpec: %v", err))
	}
	return out
}

// Delete CR; bucket persists.
func (s *ControllerExecutor) DeleteSandbox(ctx context.Context, userID int64, projectID string) error {
	err := s.dynamic.Resource(AgentSandboxGVK).
		Namespace(s.namespaceFor(userID)).
		Delete(ctx, SandboxName(projectID), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// Wait phase=Running; URL pre-allocated.
func (s *ControllerExecutor) WaitRunning(ctx context.Context, userID int64, projectID string, timeout time.Duration) error {
	res := s.dynamic.Resource(AgentSandboxGVK).Namespace(s.namespaceFor(userID))
	deadline := time.Now().Add(timeout)
	var lastPhase string
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("wait running timed out (last phase=%s)", lastPhase)
		}
		obj, err := res.Get(ctx, SandboxName(projectID), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("agentsandbox not found: %w", err)
		}
		if err != nil {
			return err
		}
		phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
		lastPhase = phase
		switch phase {
		case "Running":
			return nil
		case "Failed":
			return errors.New("sandbox phase=Failed")
		case "Expired":
			return errors.New("sandbox phase=Expired")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// Poll + return typed status snapshot.
func (s *ControllerExecutor) FetchRunning(ctx context.Context, userID int64, projectID string, timeout time.Duration) (*project.SandboxStatus, error) {
	res := s.dynamic.Resource(AgentSandboxGVK).Namespace(s.namespaceFor(userID))
	deadline := time.Now().Add(timeout)
	var lastPhase string
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("wait running timed out (last phase=%s)", lastPhase)
		}
		obj, err := res.Get(ctx, SandboxName(projectID), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("agentsandbox not found: %w", err)
		}
		if err != nil {
			return nil, err
		}
		phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
		lastPhase = phase
		switch phase {
		case "Running":
			return sandboxStatusFromUnstructured(obj)
		case "Failed":
			return sandboxStatusFromUnstructured(obj)
		case "Expired":
			return sandboxStatusFromUnstructured(obj)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// Snapshot read of latest stamped status (URL may be "" pre-reconcile).
func (s *ControllerExecutor) GetStatus(ctx context.Context, userID int64, projectID string) (*project.SandboxStatus, error) {
	res := s.dynamic.Resource(AgentSandboxGVK).Namespace(s.namespaceFor(userID))
	obj, err := res.Get(ctx, SandboxName(projectID), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sandboxStatusFromUnstructured(obj)
}

// Extract typed status from Unstructured.
func sandboxStatusFromUnstructured(obj *unstructured.Unstructured) (*project.SandboxStatus, error) {
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	publicURL, _, _ := unstructured.NestedString(obj.Object, "status", "publicURL")
	previewHost, _, _ := unstructured.NestedString(obj.Object, "status", "previewHost")
	podName, _, _ := unstructured.NestedString(obj.Object, "status", "podName")
	bucket, _, _ := unstructured.NestedString(obj.Object, "status", "bucket")
	return &project.SandboxStatus{
		Phase:       phase,
		PublicURL:   publicURL,
		PreviewHost: previewHost,
		PodName:     podName,
		Bucket:      bucket,
	}, nil
}

// Run cmd in pod via streaming ndjson.
func (c *ControllerExecutor) Exec(ctx context.Context, podName, namespace, command string, timeout time.Duration) (string, error) {
	if podName == "" {
		return "", errors.New("podName required")
	}
	if namespace == "" {
		return "", errors.New("namespace required")
	}

	body, err := json.Marshal(struct {
		Command string        `json:"command"`
		Timeout time.Duration `json:"timeout"`
	}{Command: command, Timeout: timeout})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	url := strings.TrimRight(c.execEndpoint, "/") + "/v1/sandboxes/" + namespace + "/" + podName + "/exec"

	httpCtx, cancel := context.WithTimeout(ctx, timeout+5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("X-Sandbox-Token", c.authToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("exec http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("exec http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	return c.readExecStream(resp.Body)
}

// Mirrors controller event (local copy).
type execEvent struct {
	Stream string `json:"stream"`
	Data   string `json:"data,omitempty"`
	Code   int    `json:"code,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Parse ndjson stream into merged text.
func (c *ControllerExecutor) readExecStream(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var stdout, stderr bytes.Buffer
	var exitCode int
	var streamErrMsg string
	sawExit := false

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev execEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			streamErrMsg = fmt.Sprintf("malformed event: %v", err)
			continue
		}
		switch ev.Stream {
		case "stdout":
			stdout.WriteString(ev.Data)
		case "stderr":
			stderr.WriteString(ev.Data)
		case "error":
			streamErrMsg = ev.Error
		case "exit":
			sawExit = true
			exitCode = ev.Code
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return mergeExecOutput(stdout.String(), stderr.String()), fmt.Errorf("read exec stream: %w", err)
	}

	out := mergeExecOutput(stdout.String(), stderr.String())
	if !sawExit {
		return out, errors.New("exec stream ended without exit event")
	}
	switch {
	case streamErrMsg != "":
		if exitCode == 124 {
			return out, fmt.Errorf("%w: %s", domain.ErrExecTimeout, streamErrMsg)
		}
		return out, fmt.Errorf("exec failed (exit %d): %s", exitCode, streamErrMsg)
	case exitCode != 0:
		if exitCode == 124 {
			return out, domain.ErrExecTimeout
		}
		return out, fmt.Errorf("command exited %d", exitCode)
	}
	return out, nil
}

// Status field helpers (typed reads).

func PodName(obj *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(obj.Object, "status", "podName")
	return s
}

func PublicURL(obj *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(obj.Object, "status", "publicURL")
	return s
}

func Bucket(obj *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(obj.Object, "status", "bucket")
	return s
}

// Inverse of SandboxName.
func ExtractProjectIDFromName(name string) string {
	const prefix = "sandbox-"
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	return strings.TrimPrefix(name, prefix)
}

// Parse uid from sandbox-u-* ns.
func ExtractUserIDFromNamespace(namespace string) int64 {
	const prefix = "sandbox-u-"
	if !strings.HasPrefix(namespace, prefix) {
		return 0
	}
	uid, err := strconv.ParseInt(strings.TrimPrefix(namespace, prefix), 10, 64)
	if err != nil {
		return 0
	}
	return uid
}

// Compile-time interface assertions.
var (
	_ project.SandboxManager = (*ControllerExecutor)(nil)
	_ project.CommandRunner  = (*ControllerExecutor)(nil)
)
