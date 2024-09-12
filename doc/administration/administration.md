# Cluster Administration

Oracle Cloud Native Environment systems are administered through Kubernetes.
The only reason to access a node using some other method, such as ssh or a
serial console, is when the node cannot be accessed using the cluster.

## Creating a Shell

It may be necessary to directly interact with a node to gather information for
debugging purposes or test a configuration change.  It should be rare.  To do
so, create a console on the desired node.

```
# List cluster nodes
$ kubectl get node
NAME       STATUS   ROLES           AGE    VERSION
ocne8800   Ready    control-plane   111m   v1.28.3+3.el8

# Get a shell
$ ocne cluster console --node ocne8800
sh-4.4# ls
bin  boot  dev	etc  home  hostroot  lib  lib64  media	mnt  opt  proc	root  run  sbin  srv  sys  tmp	usr  var

# The host filesystem is mounted into the console.  Chrooting to that filesystem
# gives an experience that matches logging on to the system directly
sh-4.4# cat /proc/$(pgrep kubelet)/cmdline | tr '\0' ' '
/usr/bin/kubelet --bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf --config=/var/lib/kubelet/config.yaml --address=0.0.0.0 --authorization-mode=AlwaysAllow --container-runtime-endpoint=unix:///var/run/crio/crio.sock --node-ip=192.168.124.201 --pod-infra-container-image=container-registry.oracle.com/olcne/pause:3.9 --tls-min-version=VersionTLS12 --fail-swap-on=false --container-runtime-endpoint=unix:///var/run/crio/crio.sock --runtime-request-timeout=10m --cgroup-driver=systemd

sh-4.4# ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: enp0s2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether 52:54:00:12:34:56 brd ff:ff:ff:ff:ff:ff
    inet 10.0.10.15/24 brd 10.0.10.255 scope global dynamic noprefixroute enp0s2
       valid_lft 79498sec preferred_lft 79498sec
    inet6 fec0::5054:ff:fe12:3456/64 scope site dynamic noprefixroute 
       valid_lft 86052sec preferred_lft 14052sec
    inet6 fe80::5054:ff:fe12:3456/64 scope link noprefixroute 
       valid_lft forever preferred_lft forever
...
```

The console can also be used to execute commands without having to directly
interact with the shell.

```
$ echo 'ip addr | head' | ocne cluster console --node ocne8800
INFO[2024-05-08T13:36:27-05:00] Waiting for the Kubernetes cluster to be ready: ok 
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: enp0s2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether 52:54:00:12:34:56 brd ff:ff:ff:ff:ff:ff
    inet 10.0.10.15/24 brd 10.0.10.255 scope global dynamic noprefixroute enp0s2
       valid_lft 79403sec preferred_lft 79403sec
```
