// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/util/pidlock"
)

const (
	ClusterCacheFilename = "clusters.yaml"
)

type Cluster struct {
	ClusterConfig  types.ClusterConfig `yaml:"config"`
	KubeconfigPath string              `yaml:"kubeconfig"`
}

type ClusterCache struct {
	Clusters map[string]Cluster `yaml:"clusters"`
}

func GetCache() (*ClusterCache, error) {
	dir, err := file.GetOcneDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, ClusterCacheFilename)

	pidlock.WaitFor(10 * time.Second)
	cacheBytes, err := os.ReadFile(path)
	pidlock.Drop()

	if err != nil {
		if os.IsNotExist(err) {
			return &ClusterCache{
				Clusters: map[string]Cluster{},
			}, nil
		}
		return nil, err
	}

	ret := &ClusterCache{}
	err = yaml.Unmarshal(cacheBytes, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (cc *ClusterCache) Save() error {
	dir, err := file.EnsureOcneDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, ClusterCacheFilename)
	cacheBytes, err := yaml.Marshal(cc)
	if err != nil {
		return err
	}

	pidlock.WaitFor(10 * time.Second)
	err = os.WriteFile(path, cacheBytes, 0600)
	pidlock.Drop()

	if err != nil {
		return err
	}

	return nil
}

func (cc *ClusterCache) Get(name string) *Cluster {
	c, ok := cc.Clusters[name]
	if !ok {
		return nil
	}
	return &c
}

func (cc *ClusterCache) GetAll() map[string]Cluster {
	return cc.Clusters
}

func (cc *ClusterCache) Add(clusterConfig *types.ClusterConfig, kubeconfig string) error {
	_, ok := cc.Clusters[*clusterConfig.Name]
	if ok {
		return fmt.Errorf("A cluster named %s already exists", *clusterConfig.Name)
	}
	cc.Clusters[*clusterConfig.Name] = Cluster{
		ClusterConfig:  *clusterConfig,
		KubeconfigPath: kubeconfig,
	}
	return cc.Save()
}

func (cc *ClusterCache) Delete(name string) error {
	delete(cc.Clusters, name)
	return cc.Save()
}
