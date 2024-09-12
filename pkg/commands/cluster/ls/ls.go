// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ls

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/cluster/cache"
)

func List() error {
	clusterCache, err := cache.GetCache()
	if err != nil {
		return err
	}

	for c, _ := range clusterCache.Clusters {
		fmt.Println(c)
	}
	return nil
}
