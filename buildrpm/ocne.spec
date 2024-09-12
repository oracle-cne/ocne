%define MOD_PATH github.com/oracle-cne


%global debug_package %{nil}
%global _buildhost build-ol%{?oraclelinux}-%{?_arch}.oracle.com

Name: ocne
Version: 2.0.1
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
%license LICENSE THIRD_PARTY_LICENSES.txt
/usr/bin/ocne
%{_sysconfdir}/bash_completion.d/ocne

%changelog
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
