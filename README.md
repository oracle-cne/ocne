# Oracle Cloud Native Environment Command Line Interface

The Oracle Linux Cloud Native Environment Command Line Interface (CLI) is a
tool that manages the lifecycle of Kubernetes clusters and the applications
running inside those clusters.

## Documentation

For overall documentation, see [Oracle Linux Cloud Native Environment](https://docs.oracle.com/en/operating-systems/olcne/).  

To start using the CLI, see [Quick Start for Release 2.0](https://docs.oracle.com/en/operating-systems/olcne/2.0/quickstart/intro.html).

### Building yourself

Building the CLI requires a variety of libraries and utilities.
- Go
- Helm
- pkg-config
- gpgme

On Oracle Linux 8, please enable below dnf repos:
```
sudo dnf install -y oracle-ocne-release-el8
sudo dnf config-manager --enable ol8_codeready_builder ol8_ocne
```

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
