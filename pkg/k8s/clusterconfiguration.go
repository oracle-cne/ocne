package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmv1beta4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
	kubeadmv1beta3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/oracle-cne/ocne/pkg/constants"
)

func GetKubeadmClusterConfiguration(client kubernetes.Interface) (*kubeadm.ClusterConfiguration, error) {
	cm, err := GetConfigmap(client, constants.KubeNamespace, constants.KubeCMName)
	if err != nil {
		return nil, err
	}

	clusterConfigYAML, exists := cm.Data[constants.KubeCMField]
	if !exists {
		return nil, fmt.Errorf("%s not found in %s ConfigMap", constants.KubeCMField, constants.KubeCMName)
	}

	// Decode the YAML into ClusterConfiguration struct
	scheme := runtime.NewScheme()
	kubeadmv1beta4.AddToScheme(scheme)
	kubeadmv1beta3.AddToScheme(scheme)
	decoder := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
	obj, _, err := decoder.Decode([]byte(clusterConfigYAML), nil, &kubeadmv1beta4.ClusterConfiguration{})
	if err != nil {
		return nil, fmt.Errorf("failed to decode ClusterConfiguration: %v", err)
	}

	ret := kubeadm.ClusterConfiguration{}
	switch obj.(type) {
	case *kubeadmv1beta4.ClusterConfiguration:
		err = kubeadmv1beta4.Convert_v1beta4_ClusterConfiguration_To_kubeadm_ClusterConfiguration(obj.(*kubeadmv1beta4.ClusterConfiguration), &ret, nil)
		if err != nil {
			return nil, err
		}
	case *kubeadmv1beta3.ClusterConfiguration:
		err = kubeadmv1beta3.Convert_v1beta3_ClusterConfiguration_To_kubeadm_ClusterConfiguration(obj.(*kubeadmv1beta3.ClusterConfiguration), &ret, nil)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("ClusterConfiguration had unexpected type")
	}

	return &ret, nil
}
