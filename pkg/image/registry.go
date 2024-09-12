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
