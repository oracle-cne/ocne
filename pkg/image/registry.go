// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	log "github.com/sirupsen/logrus"
)

// AddDefaultRegistries takes a list of absolute, relative (without domain), and malformed images and returns a list of absolute and malformed images.
func AddDefaultRegistries(images []string, registry string) []string {
	toReturn := make([]string, 0, len(images))
	for _, imageToChange := range images {
		if len(imageToChange) == 0 {
			continue
		}
		toAdd, err := AddDefaultRegistry(imageToChange, registry)
		if err != nil {
			log.Errorf(err.Error())
			continue
		}
		toReturn = append(toReturn, toAdd)
	}
	return toReturn
}

func AddDefaultRegistry(imageToChange string, registry string) (string, error) {
	_, err := reference.ParseNamed(imageToChange)
	toAdd := imageToChange
	if err != nil && errors.Is(err, reference.ErrNameNotCanonical) {
		registry = strings.TrimRight(registry, "/")
		imageToChange = strings.TrimLeft(imageToChange, "/")
		toAdd = registry + "/" + imageToChange
	} else if err != nil {
		tmp := fmt.Sprintf("Could not add default source registry to image name %s due to error: %s", imageToChange, err.Error())
		return "", errors.New(tmp)
	}
	return toAdd, nil
}

// ParseOstreeReference returns two strings.  First, the ostree
// reference without a tag.  Second, a reference that is usable
// by Podman compatible container runtimes.  For example,
// "ostree-unverified-image:container-registry.oracle.com/olcne/ock-ostree:1.30"
// returns "ostree-unverified-image:container-registry.oracle.com/olcne/ock-ostree"
// and container-registry.oracle.com/olcne/ock-ostree:1.30
func ParseOstreeReference(img string) (string, string, error) {
	// Important stuff is colon delimited.
	fields := strings.Split(img, ":")

	// At the very least there needs to be a registry
	// and a tag.  More, actually, but that is all checked
	// later on.
	if len(fields) < 2 {
		return "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
	}

	switch fields[0] {
	case "ostree-unverified-image", "ostree-image-signed":
		fields = fields[1:]
		switch fields[0] {
		case "registry":
			fields = fields[1:]
		case "docker":
			// strip off the "//"
			fields[0] = fields[0][2:]
		default:
			return "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
		}
	case "ostree-unverified-registry":
		fields = fields[1:]
	case "ostree-remote-image":
		if len(fields) < 3 {
			return "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
		}
		fields = fields[2:]
	default:
		return "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
	}

	// Hack the tag off the reference for the ostree image
	imgIdx := strings.LastIndex(img, ":")
	ostreeImg := img[:imgIdx]

	return ostreeImg, strings.Join(fields, ":"), nil
}

// ParseOsRegistry returns two strings.  First, the ostree
// transport.  Second, a reference without the transport
func ParseOsRegistry(osRegistry string) (string, string, error) {
	if tr, rg, ok := strings.Cut(osRegistry, ":"); ok {
		return tr, rg, nil
	} else {
		return "", "", fmt.Errorf("%s is not a valid OsRegistry reference", osRegistry)
	}
}
