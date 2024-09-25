%define MOD_PATH github.com/oracle-cne


%global debug_package %{nil}
%global _buildhost build-ol%{?oraclelinux}-%{?_arch}.oracle.com

Name: ocne
Version: 2.0.2
Release: 3%{dist}
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
Requires: bash-completion
Requires: containers-common

%description
The Oracle Cloud Native Environment command line interface manages Kubernetes
clusters and the application in them.

%prep
%setup -q

%build
export GOPATH=`pwd`/go
mkdir -p $GOPATH/src/%{MOD_PATH}
ln -s `pwd` $GOPATH/src/%{MOD_PATH}/ocne
pushd $GOPATH/src/%{MOD_PATH}/ocne

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
