// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestIsFlannelReleaseConfigBeforeNodeIPAssignment(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		want   bool
	}{
		{
			name: "legacy single stack configuration with interface",
			config: map[string]interface{}{
				"podCidr": "10.244.0.0/16",
				"flannel": map[string]interface{}{
					"args":  []interface{}{"--ip-masq", "--kube-subnet-mgr", "--iface=eth0"},
					"image": map[string]interface{}{"tag": "v0.27.4"},
				},
			},
			want: true,
		},
		{
			name: "legacy dual stack configuration without interface",
			config: map[string]interface{}{
				"podCidr":   "10.244.0.0/16",
				"podCidrv6": "fd00:10:244::/56",
				"flannel": map[string]interface{}{
					"args":  []interface{}{"--ip-masq", "--kube-subnet-mgr"},
					"image": map[string]interface{}{"tag": "v0.27.4"},
				},
			},
			want: true,
		},
		{
			name: "custom release value",
			config: map[string]interface{}{
				"podCidr":     "10.244.0.0/16",
				"customValue": "preserved",
				"flannel": map[string]interface{}{
					"args": []interface{}{
						"--ip-masq",
						"--kube-subnet-mgr",
					},
					"image": map[string]interface{}{"tag": "v0.27.4"},
				},
			},
			want: false,
		},
		{
			name: "custom flannel argument",
			config: map[string]interface{}{
				"podCidr": "10.244.0.0/16",
				"flannel": map[string]interface{}{
					"args":  []interface{}{"--ip-masq", "--kube-subnet-mgr", "--iface=eth0", "--foo=bar"},
					"image": map[string]interface{}{"tag": "v0.27.4"},
				},
			},
			want: false,
		},
		{
			name: "already updated configuration",
			config: map[string]interface{}{
				"podCidr": "10.244.0.0/16",
				"flannel": map[string]interface{}{
					"args":     []interface{}{"--ip-masq", "--kube-subnet-mgr", "--public-ip=$(NODE_IP)"},
					"extraEnv": []interface{}{},
					"image":    map[string]interface{}{"tag": "v0.27.4"},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isFlannelReleaseConfigBeforeNodeIPAssignment(test.config); got != test.want {
				t.Errorf("isFlannelReleaseConfigBeforeNodeIPAssignment() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestAddFlannelNodeIPConfigurationAddsDefaultArguments(t *testing.T) {
	config := map[string]interface{}{
		"flannel": map[string]interface{}{
			"args": []interface{}{
				"--ip-masq",
				"--kube-subnet-mgr",
			},
		},
	}

	updated, err := addFlannelNodeIPConfiguration(config)
	if err != nil {
		t.Fatalf("addFlannelNodeIPConfiguration returned an error: %v", err)
	}
	if !updated {
		t.Fatal("addFlannelNodeIPConfiguration did not report an update")
	}

	args, found, err := unstructured.NestedSlice(config, "flannel", "args")
	if err != nil || !found {
		t.Fatalf("could not read Flannel arguments: found=%t err=%v", found, err)
	}
	for _, expected := range []string{"--ip-masq", "--kube-subnet-mgr", "--public-ip=$(NODE_IP)"} {
		if !containsString(args, expected) {
			t.Fatalf("Flannel arguments did not include %s: %#v", expected, args)
		}
	}
	extraEnv, found, err := unstructured.NestedSlice(config, "flannel", "extraEnv")
	if err != nil || !found {
		t.Fatalf("could not read Flannel environment: found=%t err=%v", found, err)
	}
	if !containsEnvironmentVariable(extraEnv, "NODE_IP") {
		t.Fatalf("Flannel environment did not include NODE_IP: %#v", extraEnv)
	}
}
