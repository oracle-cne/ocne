%define MOD_PATH github.com/oracle-cne


%global debug_package %{nil}
%global _buildhost build-ol%{?oraclelinux}-%{?_arch}.oracle.com

Name: ocne
Version: 2.3.0
Release: 1%{dist}
Vendor: Oracle America
Summary: Oracle Cloud Native Environment command line interface
License: UPL 1.0
Group: Development/Tools

Source0: %{name}-%{version}.tar.bz2
BuildRequires: golang
BuildRequires: helm >= 3.13.0
BuildRequires: gpgme-devel
BuildRequires: btrfs-progs-devel
BuildRequires: device-mapper-devel
BuildRequires: libassuan-devel
BuildRequires: yq
BuildRequires: rpm-build
Requires: bash-completion
Requires: containers-common

%description
The Oracle Cloud Native Environment command line interface manages Kubernetes
clusters and the application in them.

%prep
%setup -q

%build
export GOPATH=`pwd`/go
export GOTOOLCHAIN=local
mkdir -p $GOPATH/src/%{MOD_PATH}
ln -s `pwd` $GOPATH/src/%{MOD_PATH}/ocne
pushd $GOPATH/src/%{MOD_PATH}/ocne

# Check if code changes require updates to go.mod and/or the vendor folder.
# After each step cleanup the download from "go mod tidy" and "go mod vendor,
# it causes cannot find module providing packages errors
go mod tidy
rm -rf $GOPATH/pkg
go mod vendor
rm -rf $GOPATH/pkg
if [[ -n $(git status --porcelain --untracked-files=no) ]]; then
  git status
  git diff
  echo "******************************************************************************"
  echo "* ERROR: The result of a 'go mod tidy' or 'go mod vendor' resulted           *"
  echo "* in files being modified. These changes need to be included in-tree         *"
  echo "******************************************************************************"
  exit 1
fi

# Build the CLI
make CATALOG_REPO=%{catalog_repo} cli

%install
install -m 755 -d %{buildroot}/usr/bin
install -m 755 out/$(go env GOOS)_$(go env GOARCH)/ocne %{buildroot}/usr/bin/ocne
install -m 755 -d %{buildroot}%{_sysconfdir}/bash_completion.d
%{buildroot}/usr/bin/ocne completion bash > %{buildroot}%{_sysconfdir}/bash_completion.d/ocne
chmod 755 %{buildroot}%{_sysconfdir}/bash_completion.d/ocne

%files
%license LICENSE.txt THIRD_PARTY_LICENSES.txt
/usr/bin/ocne
%{_sysconfdir}/bash_completion.d/ocne

%changelog
* Sat Jun 28 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.2.0
- Support for Kubernetes 1.32
- Native support for Oracle Linux Virtualization Manager via the olvm provider

* Wed Apr 02 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.2-5
- Add a retry for fetching the kubelet-config during cluster start to account for slow controllers

* Wed Apr 02 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.2-4
- Tolerate errors from the OCI CAPI controllers that are very short lived

* Mon Mar 31 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.2-3
- Improve the initial deployment of keepalived and nginx for virtual IP deployments

* Tue Mar 25 2025 Michael Gianatassio <michael.gianatassio@oracle.com> - 2.1.2-2
- Fix a segmentation fault when automatically uploading custom images with the oci provider

* Fri Mar 21 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.2-1
- Log errors from the OCI CAPI provider that occur during cluster start
- Image upload for oci images takes cluster configuration and defaults file into account

* Mon Mar 10 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.1-3
- Introduce support for IPv6 single stack and dual stack configurations
- Fix an issue where mirroring the embedded catalog would try to mirror invalid tags
- Fix an issue where kube-proxy configuration was incomplete for the byo and none providers
- Expand supported ostree transports
- Allow configuring the OCI configuration profile in the oci provider
- Treat Ignition and Butane warnings as errors

* Thu Mar 06 2025 Prasad Shirodkar <prasad.shirodkar@oracle.com> - 2.1.1-2
- Install Kubernetes Gateway CRDs during cluster start

* Mon Mar 03 2025 Zaid Abdulrehman <zaid.a.abdulrehman@oracle.com> - 2.1.1-1
- Enable retries when mirroring catalags to account for container registry errors

* Wed Feb 12 2025 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.1.0-1
- Introduce support for Kubernetes 1.31
- Fix an issue where in-place updates in environments without internet access can disrupt cluster function
- Add support for staging Cluster API resources with the oci provider

* Tue Nov 19 2024 George Aeillo <george.f.aeillo@oracle.com> - 2.0.5-5
- Allow YAML output from cluster dumps
- Fix various issues in cluster dump and info commands
- Add info subcommand to print basic command information

* Mon Nov 18 2024 Michael Gianatassio <michael.gianatassio@oracle.com> - 2.0.5-4
- Update Go dependencies
- Allow automatic installation of the community catalog from ArtifactHub.io

* Fri Nov 15 2024 Prasad Shirodkar <prasad.shirodkar@oracle.com> - 2.0.5-3
- Fix an issue where trying to update an application with no customizations would fail

* Thu Nov 14 2024 Zaid Abdulrehman <zaid.a.abdulrehman@oracle.com> - 2.0.5-2
- Support wider range of OCI compute shapes when using the OCI provider
- Improve output when parsing errors are encountered when parsing cluster configuration

* Wed Oct 30 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.5-1
- Allow customization of ignition provider and kernel command line during image create

* Mon Oct 28 2024 Guoyong Zhang <guoyong.zhang@oracle.com> - 2.0.4-3
- Added TLS cipher suites support

* Wed Oct 23 2024 Michael Gianatassio <michael.gianatassio@oracle.com> - 2.0.4-2
- Install Catalog and UI with custom container image registry when using the "none" provider

* Tue Oct 22 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.4-1
- Fixed an issue where Flannel-based pod networking did not function with a default oci provider configuration
- The oci provider can now automatically configure OCI-CCM
- Deleting clusters that use the oci provider is now synchronous

* Mon Oct 07 2024 Shih-Chang Chen <shih-chang.chen@oracle.com> - 2.0.3-5
- Ensure that OCI metadata endpoint is in the no_proxy settings for the oci provider

* Fri Oct 04 2024 Prasad Shirodkar <prasad.shirodkar@oracle.com> - 2.0.3-4
- Fix command line and config file precendence for cluster template

* Tue Oct 01 2024 Zaid Abdulrehman <zaid.a.abdulrehman@oracle.com> - 2.0.3-3
- Allow images to be created on clusters of different versions

* Mon Sep 30 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.3-2
- Fix command line and config file precedence for cluster delete
- Validate required OCI-CCM configurations options when using oci provider
- Increase the timeout in the keepalived livness script

* Wed Sep 25 2024 George Aeillo <george.f.aeillo@oracle.com> - 2.0.3-1
- Remove unused cluster update command
- Allow for custom container image tags for OCK images

* Mon Sep 23 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.2-3
- Fixed an issue where specifying an ostree transport for the osRegistry configuration option would misconfigure the update service
- Extended the set of supported ostree transports

* Fri Sep 20 2024 George Aeillo <george.f.aeillo@oracle.com> - 2.0.2-2
- Fixed an issue where cluster dump would omit information when performing a redacted dump
- Fixed an issue where CLI arguments were not taking precedence over cluster configuration files during cluster start

* Thu Sep 19 2024 Michael Gianatassio <michael.gianatassio@oracle.com> - 2.0.2-1
- Added -c option to application update to allow for catalog selection
- Added --reset-values option to application update to allow complete reconfiguration of an application

* Fri Sep 13 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.1-3
- Add support for double dash commands when using the cluster console

* Thu Sep 12 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.1-2
- Change the default ostree registry to include a transport
- Tolerate less well specified ostree registry references

* Wed Sep 11 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.1-1
- Remove copy of the catalog repository in favor of cloning it during build
- Improve error reporting when a kubeconfig is given but the file does not exist
- Ensure that container images used in Helm webhooks appear in the output of catalog mirror commands

* Wed Sep 11 2024 Shih-Chang Chen <shih-chang.chen@oracle.com> - 2.0.0-6
- Deploy proxy settings for rpm-ostreed.service

* Tue Sep 10 2024 Michael Gianatassio <michael.gianatassio@oracle.com> - 2.0.0-5
- Remove race condition detection from instrumented build to avoid catching races in dependencies
- Improve the coverage and reliability of integration tests
- Fix a race condition in the dump and analyze commands

* Tue Sep 10 2024 George Aeillo <george.f.aeillo@oracle.com> - 2.0.0-4
- Automatically download a boot volume when a command that uses an ephemeral cluster is the first command executed on a clean host
- Add Kubernetes version to log messages when starting a cluster

* Mon Sep 09 2024 Prasad Shirodkar <prasad.shirodkar@oracle.com> - 2.0.0-3
- Improve documentation

* Fri Sep 06 2024 Zaid Abdulrehman <zaid.a.abdulrehman@oracle.com> - 2.0.0-2
- Add a flag to automatically chroot to the node root filesystem when accessing the cluster console
- Deploy core Oracle Cloud Native Environment in-cluster services to function better in offline environments
 
* Sat Aug 31 2024 Daniel Krasinski <daniel.krasinski@oracle.com> - 2.0.0-1
- Initial release of the Oracle Cloud Native Environment CLI
