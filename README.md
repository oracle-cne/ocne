# Oracle Cloud Native Environment Command Line Interface

The Oracle Linux Cloud Native Environment Command Line Interface (CLI) is a
tool that manages the lifecycle of Kubernetes clusters and the applications
running inside those clusters.

## Installation

The CLI is available for Oracle Linux 8 and Oracle Linux 9.  It can be
installed from repositories on the [Oracle Linux YUM Repository](https://yum.oracle.com).  It can also be built from this source code.

### Oracle Linux 8

Perform the following steps to install the Oracle Cloud Native Environment
YUM repository, enable it, and install the CLI.

```
dnf install -y oracle-ocne-release-el8
dnf config-manager --enable ol8_ocne
dnf install -y ocne
```

### Oracle Linux 9

Installing the CLI on Oracle Linux 9 can be done with the following
instructions.

```
dnf install -y oracle-ocne-release-el9
dnf config-manager --enable ol9_ocne
dnf install -y ocne
```

### Building yourself

Building the CLI requires a variety of libraries and utilities.
- Go
- Helm
- pkg-config
- gpgme

These dependencies can be installed on Oracle Linux 8 and Oracle Linux 9
by leveraging `yum-buildep`.

```
yum-builddep buildrpm/ocne.spec
```

## Contributing

This project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md)

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security vulnerability disclosure process

## License

Copyright (c) 2023 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at
<https://oss.oracle.com/licenses/upl/>.
