package k8s

import (
    "fmt"

    "k8s.io/client-go/kubernetes"

    kubeadmv1beta4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
    "k8s.io/apimachinery/pkg/runtime/serializer/json"
    "k8s.io/apimachinery/pkg/runtime"

    "github.com/oracle-cne/ocne/pkg/constants"
)

func GetKubeadmClusterConfiguration(client kubernetes.Interface) (*kubeadmv1beta4.ClusterConfiguration, error) {
    cm, err := GetConfigmap(client, constants.KubeNamespace, constants.KubeCMName)
    if err != nil {
	    return nil, err
    }

    clusterConfigYAML, exists := cm.Data[constants.KubeCMField]
    if !exists {
        return nil, fmt.Errorf("ClusterConfiguration not found in kubeadm-config ConfigMap")
    }

    // Decode the YAML into ClusterConfiguration struct
    scheme := runtime.NewScheme()
    kubeadmv1beta4.AddToScheme(scheme)
    decoder := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
    obj, _, err := decoder.Decode([]byte(clusterConfigYAML), nil, &kubeadmv1beta4.ClusterConfiguration{})
    if err != nil {
        return nil, fmt.Errorf("failed to decode ClusterConfiguration: %v", err)
    }

    clusterConfig, ok := obj.(*kubeadmv1beta4.ClusterConfiguration)
    if !ok {
        return nil, fmt.Errorf("unexpected type after decoding")
    }

    return clusterConfig, nil
}
