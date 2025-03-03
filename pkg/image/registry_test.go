// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOstreeReference(t *testing.T) {
	cases := [][]string{
		[]string{"ostree-unverified-image:registry:container-registry.oracle.com/olcne/ock-ostree", "ostree-unverified-image:registry", "container-registry.oracle.com/olcne/ock-ostree", ""},
		[]string{"ostree-unverified-image:docker://container-registry.oracle.com/olcne/ock-ostree", "ostree-unverified-image:docker://", "container-registry.oracle.com/olcne/ock-ostree", ""},
		[]string{"ostree-unverified-image:registry:container-registry.oracle.com/olcne/ock-ostree:1.30", "ostree-unverified-image:registry", "container-registry.oracle.com/olcne/ock-ostree", "1.30"},
		[]string{"ostree-unverified-image:docker://container-registry.oracle.com/olcne/ock-ostree:1.30", "ostree-unverified-image:docker://", "container-registry.oracle.com/olcne/ock-ostree", "1.30"},
		[]string{"ostree-unverified-image:containers-storage:container-registry.oracle.com/olcne/ock-ostree", "ostree-unverified-image:containers-storage", "container-registry.oracle.com/olcne/ock-ostree", ""},
		[]string{"ostree-unverified-image:containers-storage:container-registry.oracle.com/olcne/ock-ostree:1.30", "ostree-unverified-image:containers-storage", "container-registry.oracle.com/olcne/ock-ostree", "1.30"},
		[]string{"ostree-unverified-image:oci-archive:/var/archive.tar.gz", "ostree-unverified-image:oci-archive", "/var/archive.tar.gz", ""},
		[]string{"ostree-unverified-registry:container-registry.oracle.com/olcne/ock-ostree", "ostree-unverified-registry", "container-registry.oracle.com/olcne/ock-ostree", ""},
		[]string{"ostree-unverified-registry:container-registry.oracle.com/olcne/ock-ostree:1.30", "ostree-unverified-registry", "container-registry.oracle.com/olcne/ock-ostree", "1.30"},
		[]string{"ostree-remote-image:remotename:docker://container-registry.oracle.com/olcne/ock-ostree", "ostree-remote-image:remotename:docker://", "container-registry.oracle.com/olcne/ock-ostree", ""},
		[]string{"ostree-remote-image:remotename:docker://container-registry.oracle.com/olcne/ock-ostree:1.30", "ostree-remote-image:remotename:docker://", "container-registry.oracle.com/olcne/ock-ostree", "1.30"},
	}

	for i, c := range cases {
		xport, ref, tag, err := ParseOstreeReference(c[0])
		assert.Equal(t, c[1], xport, "%d: %+v", i, err)
		assert.Equal(t, c[2], ref, "%d: %+v", i, err)
		assert.Equal(t, c[3], tag, "%d: %+v", i, err)
		assert.Nil(t, err)
	}
}
