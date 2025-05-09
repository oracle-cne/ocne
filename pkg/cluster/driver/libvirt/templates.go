// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

const (
	poolTemplate = ` 
<pool type='dir'>
  <name>{{.Name}}</name>
  <capacity unit='bytes'>38069878784</capacity>
  <allocation unit='bytes'>27457032192</allocation>
  <available unit='bytes'>10612846592</available>
  <source>
  </source>
  <target>
    <path>{{.Path}}</path>
    <permissions>
      <mode>0711</mode>
      <owner>0</owner>
      <group>0</group>
      <label>system_u:object_r:virt_image_t:s0</label>
    </permissions>
  </target>
</pool>
`
	volumeTemplate = ` 
<volume type='file'>
  <name>{{.Name}}</name>
  <capacity unit='{{.StorageUnit}}'>{{.Size}}</capacity>
  <target>
    <format type='qcow2'/>
    <permissions>
      <mode>0644</mode>
    </permissions>
    <compat>1.1</compat>
    <clusterSize unit='B'>65536</clusterSize>
    <features/>
  </target>
  <backingStore>
    <path>{{.PathToBackingStore}}</path>
    <format type='qcow2'/>
  </backingStore>
</volume>
`

	volumeTemplateNoBacking = `
<volume type='file'>
  <name>{{.Name}}</name>
  <capacity unit='bytes'>{{.Size}}</capacity>
  <target>
    <format type='{{.Type}}'/>
    <permissions>
      <mode>0644</mode>
    </permissions>
    <compat>1.1</compat>
    <features/>
  </target>
</volume>
`

	domainTemplate = `
<domain type='{{.Hypervisor}}' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>{{.Name}}</name>
  <description>{{.Description}}"</description>
  <memory unit='{{.MemoryCapacityUnit}}'>{{.Memory}}</memory>
  <vcpu placement='static'>{{.CPUs}}</vcpu>
  <resource>
    <partition>/machine</partition>
  </resource>
  {{if eq .CPUArch "aarch64"}}
	<os firmware="efi">
		<firmware>
			<feature enabled="no" name="enrolled-keys"/>
			<feature enabled="no" name="secure-boot"/>
		</firmware>
		<type arch='{{.CPUArch}}' machine='virt'>hvm</type>
		<boot dev='hd'/>
  {{else}}
		<os firmware='efi'>
		<type arch='{{.CPUArch}}' machine='q35'>hvm</type>
		<boot dev='hd'/>
  {{end}}
  </os>
  {{if eq .CPUArch "aarch64"}}
    <cpu mode='host-passthrough' match='exact' check='partial'>
      <model fallback='forbid'>cortex-a57</model>
    </cpu>
    <features>
  {{else}}
	<cpu mode='host-passthrough'>
	    <feature policy='disable' name='pdpe1gb'/>
	</cpu>
    <features>
		<smm state="off"/>
  {{end}}
	<acpi/>
    <apic/>
  </features>
  <devices>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='{{.VolumePool}}' volume='{{.Volume}}'/>
      <target dev='sda' bus='scsi'/>
      <alias name='scsi0-0-0-0'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'>
      <alias name='scsi0'/>
      <address type='pci' domain='0x0000' bus='0x03' slot='0x00' function='0x0'/>
    </controller>
{{range $net := .Networks}}
  {{if (eq $net.Type "network")}}
    <interface type='network'>
      <source network='{{$net.Network}}'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='{{$net.Bus}}' slot='{{$net.Slot}}' function='0x0' />
    </interface>
  {{end}}
{{end}}
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
{{if eq .CPUArch "x86_64"}}
    <input type='keyboard' bus='ps2'>
      <alias name='input1'/>
    </input>
{{end}}
    <audio id='1' type='none'/>
    <memballoon model='virtio'>
      <alias name='balloon0'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </memballoon>
    <rng model='virtio'>
      <backend model='random'>/dev/urandom</backend>
      <alias name='rng0'/>
    </rng>
  </devices>
  <qemu:commandline>
{{if .IgnitionPath}}
    <qemu:arg value='-fw_cfg'/>
    <qemu:arg value='name=opt/com.coreos/config,file={{.IgnitionPath}}'/>
{{end}}
{{range $net := .Networks}}
  {{if (eq $net.Type "user")}}
    <qemu:arg value='-netdev'/>
    <qemu:arg value='user,id=mynet,net={{$net.Subnet}}{{range $pf := $net.PortForwards}},hostfwd=tcp:{{$pf.Listen}}:{{$pf.From}}-:{{$pf.To}}{{end}}'/>
    <qemu:arg value='-device'/>
    <qemu:arg value='virtio-net,netdev=mynet,addr=0{{$net.Slot}}.0'/>
  {{end}}
{{end}}
  </qemu:commandline>
</domain>
`
)
