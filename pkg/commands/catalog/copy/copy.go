// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package copy

import (
	"bufio"
	"fmt"
	"github.com/containers/image/v5/copy"
	log "github.com/sirupsen/logrus"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/image"
	"os"
	"strings"
	"time"
)

func Copy(opt catalog.CopyOptions) error {
	imageURLs := opt.Images
	if opt.Images == nil {
		// Extract image URLs from the source file
		var err error
		imageURLs, err = extractImageURLs(opt.FilePath)
		if err != nil {
			return err
		}
	}
	var newImageURLs []string
	var originalImageURLs []string
	// Add docker:// to each image URL
	for _, theImage := range imageURLs {
		if !strings.HasPrefix(theImage, "docker://") {
			theImage = "docker://" + theImage
		}
		newImageCreated, err := image.ChangeImageDomain(theImage, opt.Destination)
		if err != nil {
			log.Warnf("Skipping invalid image %s: %v", theImage, err)
			continue
		}
		originalImageURLs = append(originalImageURLs, theImage)
		newImageURLs = append(newImageURLs, "docker://"+newImageCreated.DockerReference().String())
	}

	// Copy images to the new registry
	copiedImages, err := copyImagesToNewDomain(originalImageURLs, newImageURLs, "")
	if err != nil {
		return err
	}

	if (copiedImages != nil) && len(copiedImages) > 0 {
		log.Info("Container images have been copied from one registry to another")
	} else {
		log.Info("No images were copied")
	}

	if opt.DestinationFilePath != "" {
		// Write updated image URLs to the destination file
		err = writeUpdatedImageURLs(copiedImages, opt.Destination, opt.DestinationFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractImageURLs reads the file and extracts the image URLs as a list of strings
func extractImageURLs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	var imageURLs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			imageURLs = append(imageURLs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading source file: %w", err)
	}

	return imageURLs, nil
}

// copyImagesToNewDomain parses the image list and copies the images to the new domain returning a list of images that were copied successfully
func copyImagesToNewDomain(images []string, newImageURLs []string, arch string) ([]string, error) {
	var err error
	copied := make([]string, 0, len(images))
	for i, theimage := range images {
		newImageURL := newImageURLs[i]
		for j := 0; j < 5; j++ {
			err = image.Copy(theimage, newImageURL, arch, copy.CopyAllImages)
			if err != nil {
				log.Errorf("Error copying image %s: %s", theimage, err.Error())
			} else {
				log.Infof("Successfully copied image %s", theimage)
				copied = append(copied, theimage)
			}
			if strings.Contains(err.Error(), "500 Internal Server Error") {
				log.Debugf("Backing off and retrying pulling %s from the registry", theimage)
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}
	return copied, err
}

// writeUpdatedImageURLs writes the updated image URLs to the destination file
func writeUpdatedImageURLs(images []string, newDomain string, destinationFilePath string) error {
	file, err := os.Create(destinationFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, theimage := range images {
		imageRef, err := image.ChangeImageDomain(theimage, newDomain)
		if err != nil {
			log.Errorf("failed to write image %s to destination file, could not change domain: %v", theimage, err)
			continue
		}
		dockerRef := imageRef.DockerReference()
		if dockerRef == nil {
			log.Errorf("Failed to write image %s to destination file, could not change image to docker image", theimage)
			continue
		}
		_, err = writer.WriteString(dockerRef.String() + "\n")
		if err != nil {
			log.Errorf("failed to write to destination file: %v", err)
		}
	}

	return nil
}
