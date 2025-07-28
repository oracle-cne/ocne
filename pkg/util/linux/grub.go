// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package linux

import (
	"fmt"
	"strings"
)

type MenuEntry struct {
	Title string
	KernelArgs []string
	Kernel string
	Initrd string
}

func ParseMenuEntry(entry string) (*MenuEntry, error) {
	ret := &MenuEntry{}

	lines := strings.Lines(entry)

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("grub menu line \"%s\" is invalid", line)
		}

		switch fields[0] {
		case "title":
			ret.Title = strings.TrimPrefix(line, fmt.Sprintf("%s ", line))
		case "options":
			ret.KernelArgs = fields
		case "linux":
			ret.Kernel = fields[1]
		case "initrd":
			ret.Initrd = fields[1]
		}
	}

	// Every menu entry needs all of these
	if ret.Title == "" {
		return nil, fmt.Errorf("grub menu entry did not have a title")
	}
	if len(ret.KernelArgs) == 0 {
		return nil, fmt.Errorf("grub menu entry did not have kernel arguments")
	}
	if ret.Kernel == "" {
		return nil, fmt.Errorf("grub menu entry did not have a kernel")
	}
	if ret.Initrd == "" {
		return nil, fmt.Errorf("grub menu entry did not have an initrd")
	}

	return ret, nil
}

func GetKernelArg(args []string, arg string) []string {
	ret := []string{}
	for _, a := range args {
		// If the strings are literally equivalent,
		// return that.
		if a == arg {
			return []string{a}
		}
		prefix := fmt.Sprintf("%s=", arg)
		if !strings.HasPrefix(a, prefix) {
			continue
		}

		ret = append(ret, strings.TrimPrefix(a, prefix))


	}

	return ret
}
