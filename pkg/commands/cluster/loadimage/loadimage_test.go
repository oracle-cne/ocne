// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loadimage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/oracle-cne/ocne/pkg/image"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestLoadRejectsInvalidFileBeforeClusterAccess(t *testing.T) {
	dir := t.TempDir()
	for _, archive := range []string{filepath.Join(dir, "absent.tar"), dir} {
		if err := Load("absent-kubeconfig", "", "", archive, io.Discard, io.Discard); err == nil || strings.Contains(err.Error(), "kubeconfig") {
			t.Fatalf("expected archive error for %s, got %v", archive, err)
		}
	}
}

func TestLoadNodeSelectionAndCleanup(t *testing.T) {
	for _, tc := range []struct {
		name, node, selector, first string
		empty                       bool
	}{
		{name: "all", first: "control-plane"},
		{name: "node", node: "worker", first: "worker"},
		{name: "missing-node", node: "absent", empty: true},
		{name: "equality", selector: "pool=development", first: "worker"},
		{name: "set", selector: "pool in (development,production)", first: "control-plane"},
		{name: "existence", selector: "pool", first: "control-plane"},
		{name: "conjunction", selector: "pool=development,!disabled", first: "worker"},
		{name: "no-match", selector: "pool=absent", empty: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var pod v1.Pod
			created, deleted, execRequested := 0, 0, false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/v1/nodes":
					if got := r.URL.Query().Get("labelSelector"); got != tc.selector {
						t.Errorf("selector = %q, want %q", got, tc.selector)
					}
					selector, err := labels.Parse(tc.selector)
					if err != nil {
						t.Fatal(err)
					}
					nodes := []v1.Node{
						{ObjectMeta: metav1.ObjectMeta{Name: "control-plane", Labels: map[string]string{"pool": "production"}}, Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{Architecture: "amd64"}}},
						{ObjectMeta: metav1.ObjectMeta{Name: "worker", Labels: map[string]string{"pool": "development"}}, Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{Architecture: "amd64"}}},
					}
					filtered := &v1.NodeList{}
					for _, node := range nodes {
						if selector.Matches(labels.Set(node.Labels)) {
							filtered.Items = append(filtered.Items, node)
						}
					}
					json.NewEncoder(w).Encode(filtered)
				case r.URL.Path == "/api/v1/namespaces/ocne-system":
					json.NewEncoder(w).Encode(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ocne-system"}})
				case strings.HasSuffix(r.URL.Path, "/exec"):
					execRequested = true
					if r.URL.Query().Get("stdin") != "true" || r.URL.Query().Get("tty") == "true" {
						t.Error("archive transfer must use stdin without a terminal")
					}
					if got := strings.Join(r.URL.Query()["command"], " "); got != "chroot /hostroot podman load" {
						t.Errorf("unexpected host load command: %s", got)
					}
					http.Error(w, "controlled exec failure", http.StatusBadRequest)
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pods"):
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(body, nil, &pod); err != nil {
						t.Error(err)
						return
					}
					created++
					if pod.Spec.NodeName != tc.first {
						t.Errorf("selected node %q, want %q", pod.Spec.NodeName, tc.first)
					}
					if pod.Spec.Containers[0].TTY || (pod.Spec.Containers[0].SecurityContext.Privileged == nil || !*pod.Spec.Containers[0].SecurityContext.Privileged) {
						t.Error("loader must run with host administration access and no TTY")
					}
					json.NewEncoder(w).Encode(&pod)
				case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pods/"):
					pod.Status.ContainerStatuses = []v1.ContainerStatus{{Ready: true}}
					json.NewEncoder(w).Encode(&pod)
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/pods/"):
					deleted++
					json.NewEncoder(w).Encode(&metav1.Status{Status: "Success"})
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			dir := t.TempDir()
			kubeconfig := filepath.Join(dir, "config")
			config := "apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: " + server.URL + "\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user: {}\n"
			if err := os.WriteFile(kubeconfig, []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			source := "oci:" + imageLayout(t, "amd64") + ":test-image"
			err := Load(kubeconfig, tc.node, tc.selector, source, io.Discard, io.Discard)
			if err == nil {
				t.Fatal("expected controlled failure")
			}
			if tc.empty {
				if created != 0 || deleted != 0 || execRequested {
					t.Fatalf("empty selection must fail before creating pods: %v", err)
				}
				if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no cluster nodes match") {
					t.Fatalf("unexpected empty selection error: %v", err)
				}
			} else if created != 1 || deleted != 1 || !execRequested || !strings.Contains(err.Error(), "controlled exec failure") {
				t.Fatalf("load failure must stop traversal and remove pod: created=%d deleted=%d exec=%v error=%v", created, deleted, execRequested, err)
			}
		})
	}
}

// imageLayout builds image metadata, not executable foreign-architecture images.
// The nested index exercises the same selection used for registry manifest lists.
func imageLayout(t *testing.T, arches ...string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "blobs", "sha256"), 0700); err != nil {
		t.Fatal(err)
	}
	writeBlob := func(value interface{}, mediaType string) map[string]interface{} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if err := os.WriteFile(filepath.Join(dir, "blobs", "sha256", hash), data, 0600); err != nil {
			t.Fatal(err)
		}
		return map[string]interface{}{"mediaType": mediaType, "digest": "sha256:" + hash, "size": len(data)}
	}
	manifests := []map[string]interface{}{}
	for _, arch := range arches {
		config := writeBlob(map[string]interface{}{
			"architecture": arch, "os": "linux",
			"rootfs": map[string]interface{}{"type": "layers", "diff_ids": []string{}},
			"config": map[string]interface{}{"Labels": map[string]string{"test-architecture": arch}},
		}, "application/vnd.oci.image.config.v1+json")
		manifest := writeBlob(map[string]interface{}{
			"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json",
			"config": config, "layers": []interface{}{},
		}, "application/vnd.oci.image.manifest.v1+json")
		manifest["platform"] = map[string]string{"architecture": arch, "os": "linux"}
		manifests = append(manifests, manifest)
	}
	var selected map[string]interface{}
	if len(manifests) == 1 {
		selected = manifests[0]
	} else {
		selected = writeBlob(map[string]interface{}{
			"schemaVersion": 2, "mediaType": "application/vnd.oci.image.index.v1+json", "manifests": manifests,
		}, "application/vnd.oci.image.index.v1+json")
	}
	selected["annotations"] = map[string]string{"org.opencontainers.image.ref.name": "test-image"}
	index, err := json.Marshal(map[string]interface{}{"schemaVersion": 2, "manifests": []interface{}{selected}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), index, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oci-layout"), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStageImageSelectsNodeArchitecture(t *testing.T) {
	source := "oci:" + imageLayout(t, "amd64", "arm64") + ":test-image"
	for _, arch := range []string{"amd64", "x86_64", "arm64", "aarch64"} {
		t.Run(arch, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "image.tar")
			if err := stageImage(source, arch, dest); err != nil {
				t.Fatal(err)
			}
			ref, err := alltransports.ParseImageName("oci-archive:" + dest)
			if err != nil {
				t.Fatal(err)
			}
			img, err := ref.NewImage(context.Background(), image.GetSystemContext(arch))
			if err != nil {
				t.Fatal(err)
			}
			defer img.Close()
			config, err := img.OCIConfig(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			want := image.GetSystemContext(arch).ArchitectureChoice
			if config.Architecture != want || config.Config.Labels["test-architecture"] != want {
				t.Fatalf("copied wrong manifest: %+v", config)
			}
		})
	}
}

func TestStageImageRejectsIncompatiblePlatform(t *testing.T) {
	for _, arches := range [][]string{{"amd64"}, {"amd64", "arm64"}} {
		source := "oci:" + imageLayout(t, arches...) + ":test-image"
		for _, arch := range []string{"s390x", ""} {
			if err := stageImage(source, arch, filepath.Join(t.TempDir(), "image.tar")); err == nil {
				t.Fatalf("unsupported architecture %q must not fall back to host", arch)
			}
		}
	}
	source := "oci:" + imageLayout(t, "amd64") + ":test-image"
	if err := stageImage(source, "arm64", filepath.Join(t.TempDir(), "image.tar")); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("single image must reject an incompatible node: %v", err)
	}
	armSource := "oci:" + imageLayout(t, "arm64") + ":test-image"
	if err := stageImage(armSource, "amd64", filepath.Join(t.TempDir(), "image.tar")); err == nil {
		t.Fatal("foreign image accepted for amd64 node")
	}
}

func TestLoadArchiveFormats(t *testing.T) {
	source := "oci:" + imageLayout(t, "amd64") + ":test-image"
	for _, transport := range []string{"docker-archive", "oci-archive"} {
		t.Run(transport, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "image.tar")
			if err := image.Copy(source, transport+":"+file+":example.com/test/image:latest", "amd64", copy.CopySystemImage); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(t.TempDir(), "selected.tar")
			if err := stageImage(transport+":"+file, "amd64", dest); err != nil {
				t.Fatal(err)
			}
			// Bare archive paths must get as far as kubeconfig lookup.
			err := Load("absent-kubeconfig", "", "", file, io.Discard, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "kubeconfig") {
				t.Fatalf("bare %s not accepted: %v", transport, err)
			}
		})
	}
}

func TestLoadRejectsInvalidSelectionBeforeClusterAccess(t *testing.T) {
	for _, tc := range []struct{ node, selector, message string }{
		{selector: "pool in (", message: "invalid node selector"},
		{node: "worker", selector: "pool=development", message: "mutually exclusive"},
	} {
		err := Load("absent-kubeconfig", tc.node, tc.selector, "absent.tar", io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Fatalf("expected %s, got %v", tc.message, err)
		}
	}
}

// SplitImage parses registry identity, whereas staging must also accept unnamed
// sources whose transport has no DockerReference.
func TestStageUnnamedOCISource(t *testing.T) {
	source := "oci:" + imageLayout(t, "amd64")
	if _, err := image.SplitImage(source); err == nil {
		t.Fatal("SplitImage unexpectedly accepted an unnamed OCI source")
	}
	dest := filepath.Join(t.TempDir(), "image.tar")
	if err := stageImage(source, "amd64", dest); err != nil {
		t.Fatal(err)
	}
	ref, err := alltransports.ParseImageName("oci-archive:" + dest)
	if err != nil {
		t.Fatal(err)
	}
	img, err := ref.NewImage(context.Background(), image.GetSystemContext("amd64"))
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	config, err := img.OCIConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if config.Architecture != "amd64" {
		t.Fatalf("unexpected platform: %s", config.Architecture)
	}
}
