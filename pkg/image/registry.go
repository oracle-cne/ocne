// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"errors"
	"fmt"
	"github.com/containers/image/v5/docker/reference"
	log "github.com/sirupsen/logrus"
	"strings"
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

// ParseOstreeReference returns three strings: the ostree transport, registry,
// and tag for an ostree container image.  For example,
// "ostree-unverified-image:container-registry.oracle.com/olcne/ock-ostree:1.30"
// returns "ostree-unverified-image", "container-registry.oracle.com/olcne/ock-ostree"
// and "1.30".  It is not necessary that a tag is present, but all the rest of the
// fields are required.
func ParseOstreeReference(img string) (string, string, string, error) {
	// Important stuff is colon delimited.
	fields := strings.Split(img, ":")

	// At the very least there needs to be a transport and a registry.
	if len(fields) < 2 {
		return "", "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
	}

	ostreeTransport := fields[0]
	switch ostreeTransport {
	case "ostree-unverified-image", "ostree-image-signed":
		fields = fields[1:]
		ostreeTransport = fmt.Sprintf("%s:%s", ostreeTransport, fields[0])
		switch fields[0] {
		case "registry":
			fields = fields[1:]
		case "docker":
			// strip off the "//"
			ostreeTransport = fmt.Sprintf("%s://", ostreeTransport)
			fields = fields[1:]
			fields[0] = fields[0][2:]
		default:
			return "", "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
		}
	case "ostree-unverified-registry":
		ostreeTransport = fields[0]
		fields = fields[1:]
	case "ostree-remote-image":
		if len(fields) < 4 {
			return "", "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
		}

		ostreeTransport = fmt.Sprintf("%s:%s", ostreeTransport, fields[1])
		switch fields[2] {
		case "registry":
			ostreeTransport = fmt.Sprintf("%s:%s", ostreeTransport, fields[2])
		case "docker":
			ostreeTransport = fmt.Sprintf("%s:docker://", ostreeTransport)
			if !strings.HasPrefix(fields[3], "//") {
				return "", "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
			}
			fields[3] = fields[3][2:]
		}
		fields = fields[3:]
	default:
		return "", "", "", fmt.Errorf("%s is not a valid ostree image reference", img)
	}

	// Parse the registry reference.
	registry := fmt.Sprintf("%s", strings.Join(fields, ":"))

	imgInfo, err := SplitImage(registry)

	if err != nil {
		return "", "", "", err
	}

	return ostreeTransport, imgInfo.BaseImage, imgInfo.Tag, nil
}

// MakeOstreeReference does its level best to take a container image
// reference and turn it into an ostree reference.  If there is no ostree
// transport present, it will add "ostree-unverified-registry".
func MakeOstreeReference(img string) (string, error) {
	_, _, _, err := ParseOstreeReference(img)
	if err == nil {
		return img, nil
	}

	img = fmt.Sprintf("ostree-unverified-registry:%s", img)
	_, _, _, err = ParseOstreeReference(img)
	log.Debugf("Making reference %s", img)
	return img, err
}
