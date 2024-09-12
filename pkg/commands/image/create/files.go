// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

const (
	OciDhcpScriptPath = "oci.sh"
	OciDhcpScript     = `
#!/bin/bash
#
# Copyright (C) 2022 Oracle. All rights reserved.
#
# This program is free software; you can redistribute it and/or modify it under
# the terms of the GNU General Public License as published by the Free Software
# Foundation, version 2.  This program is distributed in the hope that it will
# be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General
# Public License for more details.  You should have received a copy of the GNU
# General Public License along with this program; if not, write to the Free
# Software Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA
# 021110-1307, USA.

oci_config() {
    /etc/oci-dhclient.sh || :
}

oci_restore() {
    :
}
`

	// This script actually gathers and sets the networking configuration.
	// It is lifted verbatim from a vanilla OL8 instance.
	OciDhclientScriptPath = "oci-dhclient.sh"
	OciDhclientScript     = `
#!/bin/bash
#
# Copyright (C) 2022 Oracle. All rights reserved.
#
# This program is free software; you can redistribute it and/or modify it under
# the terms of the GNU General Public License as published by the Free Software
# Foundation, version 2.  This program is distributed in the hope that it will
# be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General
# Public License for more details.  You should have received a copy of the GNU
# General Public License along with this program; if not, write to the Free
# Software Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA
# 021110-1307, USA.

log () {
    echo "$*"
}

log "set hostname, /etc/hosts, /etc/resolv.conf begin $(date):"

OPC_CONF="/etc/oci-hostname.conf"

# import the oci-hostname configuration info
if [ -f $OPC_CONF ]; then
    . $OPC_CONF
fi

# ER-28862654 - Add custom header text to /etc/resolv.conf
if ! grep -q "docs.cloud.oracle.com" /etc/resolv.conf; then
    sed -i '1i; Any changes made to this file will be overwritten whenever the\
; DHCP lease is renewed. To persist changes you must update the\
; /etc/oci-hostname.conf file. For more information see\
;[https://docs.cloud.oracle.com/iaas/Content/Network/Tasks/managingDHCP.htm#notes]\
;' /etc/resolv.conf
fi

function retry_command()
{

  retry_attempts=30
  retry_interval_sec=2
  while [ "$retry_attempts" -gt 0 ]; do

    command_success=true
    "$@" || { command_success=false; }
    if [ "$command_success" == false ]; then
      (( retry_attempts-- ))
      log "Error occurred running command $@. Will retry in $retry_interval_sec seconds"
      sleep $retry_interval_sec
    else
      log "Successfully executed the command $@"
      break
    fi
  done

  # Check if issue running command still existed after all retry_attempts
  if [ "$command_success" == false ]; then
    log "ERROR: failed to execute command '$@' (Retried $retry_attempts times)"
    return 1
  fi
}

#Usage: add_entries <file name> <keyword> <an array of the corresponding values for the keyword>
#We pass array by name so if the array name is 'arr', pass it as 'arr' instead of $arr
#This function can be used to add entries to files with a mapping format.
#For example, /etc/hosts has <ip> mapped to <fqdn/host alias>
#The function checks to see if a line containing the given 'keyword' is in the file
#If so, we check the given array of values against the existing values for the keyword in that line.
#Append the values specified in the array to the line if it doesn't already exist.
#If the file does not contain a line with the given keyword,
#the function will add a new line with the given keyword mapped to all values in the given array.
function add_entries()
{
    local file=${1}
    local keyword=${2}
    local values=$3[@]
    values=("${!values}")
    if ! grep -qw "^$keyword" $file; then
        log "Line with '$keyword' not found in $file"
        new_entry="$keyword"
        for value in "${values[@]}"
        do
            new_entry="$new_entry $value"
        done
        log "Adding '$new_entry' to $file"
        echo "$new_entry" >>  $file

    else
        log "Found line with '$keyword'"
        target_line=$(grep -w "^$keyword" $file)
        for value in "${values[@]}"
        do
            #First case needs spaces around $value to make sure it's not the prefix or suffix of another value
            #Second case checks if $value is at the end of the line
            if [[ $target_line == *" $value "* ]] || [[ $target_line == *" $value" ]]; then
                log "'$value' already exists in line"
            else
                log "Adding '$value' to line"
                sed -i "s/^\<$keyword\>.*$/& $value/g" $file
            fi
        done
    fi
}


# This function updates the hostname
# Arguments:
#   Arg1 --  OS version information to set hostname accordingly
#   Arg2 --  Hostname that needs to be set
function update_hostname()
{
    local os_version=${1}
    local new_host_name=${2}

    log "Updating hostname"

    # 1. run hostname command
    if [ $os_version -eq 6 ]; then
        # use short hostname for /etc/sysconfig/network
        # https://access.redhat.com/documentation/en-US/Red_Hat_Enterprise_Linux/6/html/Installation_Guide/sn-Netconfig-x86.html
        new_host_name_config="HOSTNAME=$new_host_name"
        log "Update /etc/sysconfig/network with new host name $new_host_name_config"

        if grep --quiet '^HOSTNAME=' /etc/sysconfig/network; then
            log "HOSTNAME exists in /etc/sysconfig/network. Updating its value"
            sed -i "s/^HOSTNAME=.*$/$new_host_name_config/g"  /etc/sysconfig/network
        else
            log "Adding HOSTNAME to /etc/sysconfig/network"
            echo "$new_host_name_config" >> /etc/sysconfig/network
        fi

        log "Running hostname command: hostname $new_host_name"
        hostname $new_host_name

    elif [ $os_version -eq 7 ]; then
        log "Running hostnamectl command: hostnamectl set-hostname $new_host_name" 
        hostnamectl set-hostname $new_host_name >> $log_file 2>&1
    elif [ $os_version -eq 8 ]; then
        systemctl is-active --quiet NetworkManager
        nm_status=$?
        if [ $nm_status -eq 0 ]; then
                log "Running nmcli command: nmcli general hostname $new_host_name"
                nmcli general hostname $new_host_name 
        else
                log "Running hostnamectl command: hostnamectl set-hostname $new_host_name" 
                hostnamectl set-hostname $new_host_name
        fi

    fi
}


# This function updates /etc/hosts and /etc/resolv.conf
# Arguments:
#   Arg1 -- new IP address
#   Arg2 -- new hostname of the system
function update_hosts_resolv()
{
    local new_ip_address=${1}
    local new_host_name=${2}

    # Retry params
    fqdn_success=1
    retries=12

    # Remove old entry from /etc/hosts so that we avoid getting
    # stale information from ipcalc.
    # First though, save the entries for
    # restoration if ipcalc encounters failures.
    old_vals=$(grep "^$new_ip_address" /etc/hosts)
    old_fqdn=$(echo $old_vals | awk -F " " '{print $2}')
    old_host_name=$(echo $old_vals | awk -F " " '{print $3}')
    log "Pre-existing fqdn is $old_fqdn and hostname is $old_host_name"
    # Now remove old entry
    sed -i "/^\<$new_ip_address\>.*$/d" /etc/hosts

    while [[ $fqdn_success -ne 0 ]] && [[ $retries -ne 0 ]]; do
        # Get fqdn
        fqdn=$(retry_command ipcalc -h $new_ip_address)
        if [[ "$fqdn" != *.*.*.*.* ]]; then
                # DNS can take upto 120 seconds to return info. Retry as long.
                sleep 10
                (( retries-- ))
        else
                fqdn_success=0
        fi
    done

    if [ $fqdn_success -ne 0 ]; then
        log "ERROR: ipcalc unsuccessful despite multiple retries for 120 seconds"
	 # Restore previously existing hostname entry but first:
        # Check for existing exact matches of hostname and delete them, if any.
        sed -i -e "/[[:space:]]\<$old_host_name\>[[:space:]\.]/d; /[[:space:]]\<$old_host_name\>$/d" /etc/hosts
	fqdn=${fqdn#HOSTNAME=}
	echo "fqdn=$fqdn" >> $log_file


        old_host_values=("$old_fqdn" "$old_host_name")
        
	# Pre-existing FQDN is null for the first boot. 
	if [ -z "$old_fqdn" ]; then
            echo "Pre-existing fqdn is  . Skip updating /etc/hosts"  >> $log_file
        else
            add_entries "/etc/hosts" "$new_ip_address" old_host_values
	fi

    else

        # ipcalc returns HOSTNAME=xxxx, need to remove "HOSTNAME="
        fqdn=${fqdn#HOSTNAME=}

        # get subnet_domain_name
        subnet_domain_name=${fqdn#$new_host_name.}

        # verify that the subnet domain is valid, we expect it is of the
        # form <subnet-name>.<vcn-name>.<oraclevcn>.<com>
        if [[ "$subnet_domain_name" != *.*.*.* ]]; then
            log "ERROR: invalid subnet domain name '$subnet_domain_name'."
        else
            # get vcn domain name - everything after the first dot in the subnet domain name
            vcn_domain_name=${subnet_domain_name#*.}
            log "fqdn=$fqdn"
            log "subnet_domain_name=$subnet_domain_name"
            log "vcn_domain_name=$vcn_domain_name"

            # 2. Update /etc/hosts if needed
            # Check for existing exact matches of hostname and delete them, if any.
            sed -i -e "/[[:space:]]\<$new_host_name\>[[:space:]\.]/d; /[[:space:]]\<$new_host_name\>$/d" /etc/hosts

            new_host_values=("$fqdn" "$new_host_name")
            # Pass array by name
            add_entries "/etc/hosts" "$new_ip_address" new_host_values

            # 3. Update /etc/resolv.conf
            # This is a temp fix till we have a resolution for a proper dhcp response
            new_search_domains=("$subnet_domain_name" "$vcn_domain_name")
            add_entries "/etc/resolv.conf" "search" new_search_domains
            echo -e "[main]\ndns = none\n" > /run/NetworkManager/conf.d/10-oci-dhclient.conf
        fi
    fi
}

# This function updates /etc/resolv.conf
# Arguments:
#   Arg1 -- new IP address
#   Arg2 -- new hostname of the system
function update_resolv()
{
    local new_ip_address=${1}
    local new_host_name=${2}

    # Retry params.
    retries=12
    fqdn_success=1

    while [[ $fqdn_success -ne 0 ]] && [[ $retries -ne 0 ]]; do
    	# Since the hostname might have been changed in /etc/hosts and we're not
    	# updating it, only using ipcalc might give us stale hostname information.
    	# To get the DNS provided hostname, use host and compare it with ipcalc's
    	# generated version. If they don't match use the one from host.
    	host_name=$(retry_command host $new_ip_address | awk -F " " '{print $5}')
    	ipcalc_name=$(retry_command ipcalc -h $new_ip_address)
    	if [[ "$ipcalc_name" != *.*.*.*.* ]] || [[ "$host_name" != *.*.*.*.* ]]; then
                # DNS can take upto 120 seconds to return info. Retry as long.
                sleep 10
                (( retries-- ))
        else
                fqdn_success=0
        fi 
    done

    if [[ $fqdn_success -ne 0 ]]; then
        log "ERROR: ipcalc and host commands failed " \
        log "despite multiple retries for 120 seconds."
    else

        # ipcalc returns HOSTNAME=xxxx, need to remove "HOSTNAME="
        ipcalc_name=${ipcalc_name#HOSTNAME=}
        log "ipcalc returned hostname: $ipcalc_name"

        # host substring will have a "." at the end. Drop it
        host_name=${host_name%?}
        log "host returned hostname: $host_name"

        use_ipcalc_hostname=0
        if [ "$host_name" = "$ipcalc_name" ]; then
            # either one will do
            fqdn=$ipcalc_name
        else
            # Likely that ipcalc has a user changed hostname.
            # This will not match the new_host_name one.
            # Use the one from host. But before that confirm
            # that host has returned a valid name.
            if [[ "$host_name" != *.*.*.*.* ]]; then
                log "Invalid hostname $host_name from host command"
                fqdn=$ipcalc_name

                # Need an additional check here for the host_name
                # being identical to the new_host_name as the user
                # could have changed the /etc/host host_name
                # entry which would not match the dhclient one.
                ipcalc_host_name=$(echo $ipcalc_name | awk -F "." '{print $1}')
                if [[ "$ipcalc_host_name" != "$new_host_name" ]]; then
                    log "ipcalc returned host $ipcalc_host_name"
                    log "dhclient returned host $new_host_name"
                    log "Using ipcalc returned hostname"
                    use_ipcalc_hostname=1
                fi
            else
                fqdn=$host_name
            fi
        fi

        # get subnet_domain_name
        if [[ "$use_ipcalc_hostname" -eq 1 ]]; then
            subnet_domain_name=${fqdn#$ipcalc_host_name.}
        else
            subnet_domain_name=${fqdn#$new_host_name.}
        fi

        # verify that the subnet domain is valid, we expect it is of the
        # form <subnet-name>.<vcn-name>.<oraclevcn>.<com>
        if [[ $subnet_domain_name != *.*.*.* ]]; then
            log "WARNING: invalid subnet domain name '$subnet_domain_name' seen."
        else
            # get vcn domain name - everything after the first dot in the subnet domain name
            vcn_domain_name=${subnet_domain_name#*.}
            log "fqdn=$fqdn"
            log "subnet_domain_name=$subnet_domain_name"
            log "vcn_domain_name=$vcn_domain_name"

            # Update /etc/resolv.conf
            new_search_domains=("$subnet_domain_name" "$vcn_domain_name")
            add_entries "/etc/resolv.conf" "search" new_search_domains
            echo -e "[main]\ndns = none\n" > /run/NetworkManager/conf.d/10-oci-dhclient.conf
        fi
    fi
}

# This function adds NM_CONTROLLED=no entry to the primary interface config file
# So that network manger does not take cotrol when installed.
# Arguments:
# Arg1 -- primary_ip

function disable_NMcontrol()
{
    local primary_ip=${1}

    # find the primary interface
    primary_if=$(ifconfig | grep -B1 $primary_ip | head -n1 | awk -F '[: ]' '{print $1}')

    # generate the primary interface's ifconfig filepath.
    cfg_file="/etc/sysconfig/network-scripts/ifcfg-${primary_if}"

    # check if the file is present.
    if [ ! -f $cfg_file ]; then
        log "$cfg_file not found, skip NM_CONTROLLED setting."
        return
    fi

    # check if the keyword is present or not
    if ! grep -qw "^NM_CONTROLLED" $cfg_file; then
        # append the line..
        echo "NM_CONTROLLED=no" >> $cfg_file
    else
       # modify the line
       sed -i "s/^\<NM_CONTROLLED\>.*$/NM_CONTROLLED=no/g" $cfg_file
    fi
}

os_version=0
if [ -f /etc/os-release ]; then
    os_string=$(grep -w VERSION /etc/os-release | awk -F "\"" '{print $2}')
    log "INFO: Obtained $os_string from /etc/os-release" 
    if [[ "$os_string" == "8."* || "$os_string" == "8"* ]]; then
        os_version=8
    elif [[ "$os_string" == "7."* || "$os_string" == "7"* ]]; then
        os_version=7
    elif [[ "$os_string" == "6."* ]]; then
        os_version=6
    fi
fi

if [ $os_version == 0 ]; then
    log "INFO: Getting OS version via uname -mrs"
    kernel_version=$(uname -mrs)
    if [[ "$kernel_version" == *"el8"* ]]; then
        os_version=8
    elif [[ "$kernel_version" == *"el7"* ]]; then
        os_version=7
    elif [[ "$kernel_version" == *"el6"* ]]; then
        os_version=6
    fi
fi

if [ $os_version == 0 ]; then
    log "ERROR: Could not obtain valid OS version. Exiting.."
    exit 1
fi

# OL8 use NetworkManager. DHCP does not provide a reason.
if [[ "$os_version" -ne 8 ]]; then
    log "$(date): Script Reason:  $reason"
else
    log "$(date): Script Reason:  NM-controlled system"
fi

# For non-OL8 OSs get the primary vnic ip only if interface has been 
# initialized.
# Ref:  https://www.isc.org/wp-content/uploads/2018/02/dhcp44cscript.html#PREINIT
if [ "$os_version" == 8 -o "$reason" != "PREINIT" ]; then
    primary_ip=$(retry_command curl -H "Authorization: Bearer Oracle" http://169.254.169.254/opc/v2/vnics/ -sf  | jq -r '.[0] | .privateIp')
    log "$(date): Primary IP obtained: $primary_ip"
fi

# This script is invoked whenever dhclient is run.
# For non-ol8 instances, We want to skip hostname update if
# $new_ip_address != $primary_ip
# so we don't run this for all interfaces
# For ol8, with the use of NM, we don't get information on the
# new_ip_address, therefore we just use the primary_ip each time.
if [ -z "$primary_ip" ]; then
    log "Skip updating hostname because primary ip is empty."
elif [[ "$os_version" -ne 8 ]] && [[ "$new_ip_address" != "$primary_ip" ]]; then
    log "Skip updating hostname because this was not invoked for the primary vnic"
else
    if [ $os_version -ne 8 ]; then
        # For non-ol8, add NM_Controlled="no" to primary network interface
        # configuration file
        disable_NMcontrol $primary_ip

        # reason why this hook was invoked. It is set by dhclient script when
        # the OS is non-ol8
        log "reason=$reason"

        # https://linux.die.net/man/8/dhclient-script
        if [ "$reason" = "BOUND" ] || [ "$reason" = "RENEW" ] || [ "$reason" = "REBIND" ] || [ "$reason" = "REBOOT" ]; then
            log "os version = $os_version"
            #These variables are set by dhclient script
            log "new_ip_address=$new_ip_address"
            log "new_host_name=$new_host_name"
            log "new_domain_name=$new_domain_name"
        else
            log "Not updating because reason=$reason"
        fi
    fi

    if [[ $PRESERVE_HOSTINFO -eq 2 ]]; then
        log "Skip updating hostname, /etc/hosts and /etc/resolv.conf"
        log "as per PRESERVE_HOSTINFO=${PRESERVE_HOSTINFO} setting"
        return 0
    fi

     #Retrieve hostname from metadata if its empty
     if [ -z $new_host_name ]; then
         new_host_name=$(retry_command curl -sf -H "Authorization: Bearer Oracle" http://169.254.169.254/opc/v2/instance/ | jq '.hostname' -r)
     fi
    
     # Will get retrieved for ol8 since we do not have DHCP provided vars.
     if [ -z $new_ip_address ]; then
         new_ip_address=$(retry_command curl -H "Authorization: Bearer Oracle" http://169.254.169.254/opc/v2/vnics/ -sf  | jq -r '.[0] | .privateIp')
     fi
    
     if [ -z $new_host_name ]; then
         echo "ERROR: new_host_name is empty after retrieving it from metadata json. Exiting."
         exit_status=1
     else
         log "new_ip_address=$primary_ip"
         if [[ $PRESERVE_HOSTINFO -eq 0 ]]; then
             # update the hostname with new hostname
             update_hostname $os_version $new_host_name
             # update hosts and resolv conf files
             update_hosts_resolv $new_ip_address $new_host_name
         elif [[ $PRESERVE_HOSTINFO -eq 1 ]]; then
             log "Skip updating hostname as per"
             log "PRESERVE_HOSTINFO=${PRESERVE_HOSTINFO} setting"
             # update hosts and resolv conf files
             update_hosts_resolv $new_ip_address $new_host_name
         elif [[ $PRESERVE_HOSTINFO -eq 3 ]]; then
             log "Skip updating hostname and /etc/hosts as per"
             log "PRESERVE_HOSTINFO=${PRESERVE_HOSTINFO} setting"
             log "Updating subnet in /etc/resolv"
             # update resolv conf file alone
             update_resolv $new_ip_address $new_host_name
        fi
    fi
fi
log "sethostname, /etc/hosts, /etc/resolv.conf END"
`

	// OCI requires custom scripting to set the hostname, as well
	// set other networking parameters.  These are gathered from
	// the instance metadata.
	//
	// Note: this is lifted verbatim from a regular OL instance.
	//  dhclientPath = "/etc/NetworkManager/dispatcher.d/11-dhclient"
	OciDhclientPath = "11-dhclient"
	OciDhclient     = `#!/bin/bash
# run dhclient.d scripts in an emulated environment

PATH=/bin:/usr/bin:/sbin
ETCDIR=/etc/dhcp
SAVEDIR=/var/lib/dhclient
interface=$1

for optname in "${!DHCP4_@}"; do
    newoptname=${optname,,};
    newoptname=new_${newoptname#dhcp4_};
    export "${newoptname}"="${!optname}";
done

[ -f /etc/sysconfig/network ] && . /etc/sysconfig/network

[ -f /etc/sysconfig/network-scripts/ifcfg-"${interface}" ] && \
    . /etc/sysconfig/network-scripts/ifcfg-"${interface}"

if [ -d $ETCDIR/dhclient.d ]; then
    for f in $ETCDIR/dhclient.d/*.sh; do
	if [ -x "${f}" ]; then
	    subsystem="${f%.sh}"
	    subsystem="${subsystem##*/}"
	    . "${f}"
	    if [ "$2" = "up" ]; then
	        "${subsystem}_config"
	    elif [ "$2" = "dhcp4-change" ]; then
	        if [ "$subsystem" = "chrony" -o "$subsystem" = "ntp" ]; then
	            "${subsystem}_config"
	        fi
	    elif [ "$2" = "down" ]; then
	        "${subsystem}_restore"
	    fi
	fi
    done
fi
`
)

func ociFiles() map[string]string {
	return map[string]string{
		OciDhclientScriptPath: OciDhclientScript,
		OciDhcpScriptPath:     OciDhcpScript,
	}
}
