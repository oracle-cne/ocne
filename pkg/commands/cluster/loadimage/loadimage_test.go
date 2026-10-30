// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loadimage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestLoadRejectsInvalidFileBeforeClusterAccess(t *testing.T) {
	dir := t.TempDir()
	for _, archive := range []string{filepath.Join(dir, "absent.tar"), dir} {
		if err := Load("absent-kubeconfig", "", archive, io.Discard, io.Discard); err == nil || strings.Contains(err.Error(), "kubeconfig") {
			t.Fatalf("expected archive error for %s, got %v", archive, err)
		}
	}
}

func TestLoadNodeSelectionAndCleanup(t *testing.T) {
	for _, selected := range []string{"", "worker", "absent"} {
		t.Run("node="+selected, func(t *testing.T) {
			var pod v1.Pod
			created, deleted, execRequested := 0, 0, false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/v1/nodes":
					json.NewEncoder(w).Encode(&v1.NodeList{Items: []v1.Node{
						{ObjectMeta: metav1.ObjectMeta{Name: "control-plane"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "worker"}},
					}})
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
					expected := selected
					if expected == "" {
						expected = "control-plane"
					}
					if pod.Spec.NodeName != expected {
						t.Errorf("selected node %q, want %q", pod.Spec.NodeName, expected)
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
			archive := filepath.Join(dir, "image.tar")
			if err := os.WriteFile(archive, []byte("controlled archive"), 0600); err != nil {
				t.Fatal(err)
			}
			err := Load(kubeconfig, selected, archive, io.Discard, io.Discard)
			if err == nil {
				t.Fatal("expected controlled failure")
			}
			if selected == "absent" {
				if created != 0 || !strings.Contains(err.Error(), "not found") {
					t.Fatalf("unknown node must fail before creating pods: %v", err)
				}
			} else if created != 1 || deleted != 1 || !execRequested {
				t.Fatalf("load failure must stop traversal and remove pod: created=%d deleted=%d exec=%v error=%v", created, deleted, execRequested, err)
			}
		})
	}
}
