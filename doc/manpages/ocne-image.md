OCNE-IMAGE "JULY 2024" Linux "User Manuals"
===========================================

NAME
----

ocne image - Interact with images of various formats

SYNOPSIS
--------

`ocne` `image` *subcommand*

DESCRIPTION
-----------

`ocne` `image` is a set of subcommands that deal with images of various formats.
It converts container images, virtual machine images, and cloud images between
one another and can migrate the results to appropriate storage.

OPTIONS
-------

`-k`, `--kubeconfig` *path*
  A Kubernetes client configuration file that describes the target cluster as
  well as how to access it.  If this option is specified, all operations that
  work against an existing Kubernetes cluster will use this cluster.  This
  option takes precedence over the `KUBECONFIG` environment variable describe
  later in this document.

ENVIRONMENT
-----------

`KUBECONFIG`
  Behaves the same way as the `--kubeconfig` option.

SUBCOMMANDS
-----------

`create` [OPTIONS]...
  Creates an image.  The source and format of the image are determined by the
  given type.  If the type is `oci`, a bootable container image is converted
  to a qcow2 image with the customizations that are required for use as a
  custom compute image in Oracle Cloud Infrastructure (OCI).  If the type is
  `ostree`, the ostree update container image is converted into an ostree
  archive server that can be used as a source of root filesystem content for a
  custom installation.

`-t`, `--type` *type*
    The type of image to create.  One of "oci" or "ostree".  The default is "oci"

`-a`, `--arch` *architecture*
    The architecture of the image to create.  One of "arm64" or "amd64".

`-v`, `--version` *version*
    The version of Kubernetes. This is the major and minor Kubernetes version.
    For example: 1.28
    If the version is not provided, then it defaults to the latest version known to the CLI.

`upload` [OPTIONS]...
  Upload an image to object storage for a specific provider, such as OCI.

`-t`, `--type` *type*
    The type of image to create.  One of "oci" or "ostree". The default is "oci". [required]

`-a`, `--arch` *architecture*
    The architecture of the image to create. One of "arm64" or "amd64".

`-f`, `--file` *path*
    The path to the image file to upload. [required]

`-b`, `--bucket` *bucket*
    The name of an OCI Object Storage bucket.  Images are uploaded to an object
    in this bucket.  Only applies to "oci" images.  The default value is
    "ocne-images".

`-c`, `--compartment` *compartment*
    An OCI Compartment.  The value can be either the fully qualified path or
    OCID of the compartment.  If a path is provided, it will be translated to
    the correct OCID.  Uploaded images are imported to this compartment as
    custom compute images.  Only applies to "oci" images.

`-p`, `--profile` *profile*
    The name of the OCI configuration profile to use when configuring OCI API
    clients.  The default value is "DEFAULT".

`-i`, `--image-name` *name*
    The name of the OCI custom compute image that is imported during upload.
    The default is "ock".

`-v`, `--version` *version*
    The version of Kubernetes embedded in the image.  Only applies to "oci"
    images.

`-d`, `--destination` *transport:format*
    An Open Container Initiative transport and target.  Container images are
    copied to this location.  See `containers-transports(5)` for a complete
    list of transports and formats.  Only applies to "ostree" images.

SEE ALSO
--------

ocne-defaults.yaml(5) containers-transports(5)

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>
