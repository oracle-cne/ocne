OCNE-APPLICATION "FEBRUARY 2024" Linux "User Manuals"
=====================================================

NAME
----

ocne application - Interact with applications in an application catalog

SYNOPSIS
--------

`ocne` `application` *subcommand*

DESCRIPTION
-----------

`ocne` `application` is a set of subcommands that deal with the lifecycle
of applications within a Kubernetes cluster.  Please refer to individual
subcommands for details of what this set of commands can do.

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

`EDITOR`
  The default document editor.

SUBCOMMANDS
-----------

`install` [OPTIONS]...
  Install an application from an application catalog or install the built-in catalog.

`-c`, `--catalog` *catalog*
    The name of the catalog that contains the application.

`-b`, `--built-in-catalog` *built-in-catalog*
    Install the built-in catalog into the ocne-system namespace.  The cluster container runtime must be configured with the image registry name..

`-N`, `--name` *application-name*
    The name of the application to install.

`-v`, `--version` *version*
    The version of the application to install.  By default, the version is
    the latest stable version of the application.

`-n`, `--namespace` *namespace*
    The Kubernetes namespace that the application is installed into.  The
    namespace is created if it does not already exist.  If this value is
    not provided, the namespace from the current context of the kubeconfig
    is used.

`-r`, `--release` *release-name*
    The name of the release of the application.  The same application can
    be installed multiple times, differentiated by release name

`-u`, `--values` *URI*
    URI of an application configuration.  The format of the configuration
    depends on the style of application served by the target catalog.  In
    general, it will be a set of Helm values.

`template` [OPTIONS]...
  Generate a documented template containing all configuration options available
  for a particular application.  The format of the template is depends on the
  style of application served by the target catalog.  In general, it will be a
  set of Helm values.

`-c`, `--catalog` *catalog*
    The name of the catalog that contains the application.

`-n`, `--name` *application-name*
    The name of the application to templatize. [required]

`-v`, `--version` *version*
    The version of the application to templatize.

`-i`, `--interactive`
    Opens the application defined by the `EDITOR` environment variable and
    populates it with the template.

`update` [OPTIONS]...
  Update an application that was deployed from an application catalog or update the built-in catalog.

`-b`, `--built-in-catalog` *built-in-catalog*
    Update the built-in catalog in the ocne-system namespace.

`-c`, `--catalog` *catalog*
The name of the catalog that contains the application.

`-r`, `--release` *release-name*
    The name of the release of the application.

`-n`, `--namespace` *namespace*
    The Kubernetes namespace that the application is installed into.  If this
    value is not provided, the namespace from the current context of the
    kubeconfig is used.

`-v`, `--version` *version*
    The version of the application to update to.  By default, the version
    is the latest stable version of the application.

`-u`, `--values` *URI*
    URI of an application configuration.  The format of the configuration
    is the same as the format used when installing the application.  In
    general, it will be a set of Helm values.

`uninstall` [OPTIONS]...
  Uninstall an application.

`-e`, `--release` *release-name*
    The name of the release of the application. [required]

`-n`, `--namespace` *namespace*
    The Kubernetes namespace that the application is installed into.  If this
    value is not provided, the namespace from the current context of the
    kubeconfig is used.

`list` [OPTIONS]...
  List applications that are installed in a Kubernetes cluster.

`-n`, `--namespace` *namespace*
    The Kubernetes namespace with applications to list.  If this value is not
    provided, the namespace from the current context of the kubeconfig is used.

`-A`, `--all`
    List applications in all Kubernetes namespaces.

`show` `--release` *release-name* [OPTIONS]...
  Show details about a particular application installed into a Kubernetes
  cluster.

`-r`, `--release` *release-name*
    The name of the release of the application.

`-n`, `--namespace` *namespace*
    The Kubernetes namespace that the application is installed into.  If this
    value is not provided, the namespace from the current context of the
    kubeconfig is used.

`-c`, `--computed`
    Show the complete configuration for the application.  The displayed
    configuration includes both the custom values and the default values.

`-d`, `--difference`
    Show the computed values and the default values for an application
    separately.

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>
