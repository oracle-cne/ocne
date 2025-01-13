OCNE-CATALOG 1 "FEBRUARY 2024" Linux "User Manuals"
===================================================

NAME
----

ocne catalog - Manage application catalogs in the Oracle CLoud Native Environment

SYNOPSIS
--------

`ocne` `catalog` *subcommand*

DESCRIPTION
-----------

`ocne` `catalog` manages the lifecycle of application catalogs with a Kubernetes
cluster.  This includes adding and removing catalogs.

OPTIONS
-------

`--kubeconfig` *path*
  A Kubernetes client configuration file that describes the target cluster as
  well as how to access it.  If this option is specified, all operations that
  work against an existing Kubernetes cluster will use this cluster.  This
  option takes precedence over the `KUBECONFIG` environment variable describe
  later in this document.

ENVIRONMENT
-----------

`-k`, `KUBECONFIG`
  Behaves the same way as the `--kubeconfig` option.

SUBCOMMANDS
-----------

`add` [OPTIONS]...
  Adds a catalog to a Kubernetes cluster.

`-N`, `--name` *name*
    The name of the added catalog. [required]

`-u`, `--uri` *URI*
    The URI of the application catalog to add. [required]

`-n`, `--namespace` *namespace*
    The namespace that the chosen catalog should be installed into

`-p`, `--protocol` *string*
    The protocol of the application catalog to add.

`remove` [OPTIONS]...
  Removes a catalog from a Kubernetes cluster.

`-N`, `--name` *name* 
    The name of the catalog to remove. [required]

`-n`, `--namespace` *namespace*
    The namespace of the catalog to remove.

`list`
  Lists the application catalogs configured for a particular Kubernetes cluster.

`get` [OPTIONS]...
  Emit a YAML document that contains the complete description of the given
  application catalog.

  The schema is as follows:
  ```
  name: *name-of-the-catalog*
  uri: *uri-of-the-catalog*
  ```
`-N`, `--name` *name*
    The name of the catalog to get. [required]

`search` [OPTIONS]...
  Discover applications in a catalog.

`-N`, `--name` *name*
    The name of the catalog to search

`-p`, `--pattern` *pattern*
    The terms to search for.  The patterns must be a valid RE2 regular
    expression.

`mirror` [OPTIONS]...
  List the known container images used by applications in an application catalog and
  optionally push them to a private registry or optionally download them to a .tgz file. 
  Images not present in the following Kubernetes objects in an application's chart should be denoted; Job, Cronjob, Pod, 
  Podtemplate, Deployment, Statefulset, Replicaset, and Replicationcontroller. 
  To denote such images, add comments in the following format to any object in the application's chart: ``# extra-image: <image>``.

`-N`, `--name` *name*
    The name of the catalog to mirror.

`-d`, `--destination` *URI*
    The URI of the destination container registry.

`-s`, `--source` *URI*
    The source registry to use for images without a registry. By default, this value is container-registry.oracle.com
    For example, olcne/ui becomes container-registry.oracle.com/olcne/ui

`-c`, `--config` *URI*
    The URI of an Oracle Cloud Native Environment configuration file.
    If a configuration file is provided, only the applications listed
    in that file are mirrored.

`-p`, `--push`
    If specified, push images from the mirrored applications to the destination.

`-q`, `--quiet`
    If specified, output only image names and omit all other output. Useful for scripting.

`-o`, `--download`
    If specified, download the images into a .tgz file saved locally. Useful for working in air-gapped environments.

`-a`, `--archive`
    If specified, set the file path of the .tgz archive of the downloaded images. Useful for working in air-gapped environments.

`copy` [OPTIONS]...
  Copy images from one location to another. The source file should contain a list of images delimited by the newline character.
  The destination should be a valid domain pointing to a container registry. The copy command copies each image from the source file
  to the destination and writes the new image uri to the file specified by the destinationfilepath.

`-f`, `--filepath` *filepath*
    The name of the source file populated with the original container images.

`-e`, `--destinationfilepath` *destinationfilepath*
    The name of the file to contain new container images.

`-d`, `--destination` *destination*
    The new domain name of the images. For example, container-registry.oracle.com is a valid destination.

SEE ALSO
--------

ocne-application(1) ocne-config.yaml(5)

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>`
