# Integration Tests

An integration test suite lives under `./test`.  These tests represent a mostly
comprehensive set of tests to flex the `ocne` code.

## Requirements

The following software must be installed:
* bats
* make
* go

The following software is optional, but useful:
* GNU parallel

## Running the Tests

There are two flavors of test: those that run in a cluster on your local system,
and those that target remote systems.  By default, only local tests will execute.

```
make integration-test
```

To enable remote tests, set the session URI for the target system:

```
export OCNE_LIBVIRT_URI=qemu+ssh://myuser@myhost/system
make integration-test
```

Most of the time, it is only necessary to run the tests against a single
deployment.  The behavior of most subcommands is equivalent regardless of how
a cluster was deployed.  It is possible to run a subset of the tests by
providing a pattern that includes only the tests that match the pattern

```
make TEST_PATTERN=remote integration-test
```


## Adding Tests

Tests can be added by creating directories and/or files under `./test/bats`.
All tests must be implemented using bats.

## Adding Deployments

New test fixtures (read: cluster deployments) are created by adding a file
under `./test/setups/<my-fixture-name>/setup`.  This file must contain exactly
two functions: `setup_suite()` and `teardown_suite()`.  The setup function
should deploy a cluster, while a the teardown function should tear it down.
