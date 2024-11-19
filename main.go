// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"github.com/oracle-cne/ocne/pkg/cluster/driver/none"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/olvm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"

	"github.com/oracle-cne/ocne/cmd/root"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/byo"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/capi"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/libvirt"
	"github.com/spf13/pflag"
)

func registerDrivers() {
	driver.RegisterDriver(byo.DriverName, byo.CreateDriver)
	driver.RegisterDriver(capi.DriverName, capi.CreateDriver)
	driver.RegisterDriver(olvm.DriverName, olvm.CreateDriver)
	driver.RegisterDriver(libvirt.DriverName, libvirt.CreateDriver)
	driver.RegisterDriver(none.DriverName, none.CreateDriver)
}

func main() {
	// Allow timestamps for logging
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	// Allow prefix matching to minimize typing
	cobra.EnablePrefixMatching = true

	// Register any cluster drivers
	registerDrivers()

	flags := pflag.NewFlagSet("ocne", pflag.ExitOnError)
	pflag.CommandLine = flags

	rootCmd := root.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
