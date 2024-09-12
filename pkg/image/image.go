// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	log "github.com/sirupsen/logrus"
	"github.com/oracle-cne/ocne/pkg/constants"
)

// imageDirectoryPrefix is used for testing to avoid writing to the users
// home directory.
var imageDirectoryPrefix = ""
var archMap = map[string]string{
	"aarch64": "arm64",
	"arm64":   "arm64",
	"x86_64":  "amd64",
	"amd64":   "amd64",
	"":        "",
}

// EnsureBaseQcow2Image ensures that the base image is downloaded. Return a tar stream of the image.
func EnsureBaseQcow2Image(imageName string, arch string) (*tar.Reader, func(), error) {
	// Fetch the image, or use the local cache if it exists
	imgRef, err := GetOrPull(imageName, arch)
	if err != nil {
		return nil, nil, err
	}

	// Assume that the image has a single layer, and that the
	// image to transfer is in that layer.
	layers, err := GetImageLayers(imgRef, arch)
	if err != nil {
		return nil, nil, err
	}

	tarStream, closer, err := GetTarFromLayerById(imgRef, layers[0].Digest.Encoded())
	if err != nil {
		return nil, nil, err
	}

	err = AdvanceTarToPath(tarStream, "disk/boot.qcow2")
	if err != nil {
		closer()
		return nil, nil, err
	}
	return tarStream, closer, nil
}

// imageToDirectory takes the name of a container image and
// translates it to something that can be used as a path
// to a directory on a filesystem.
func imageToDirectory(imageName string) string {
	prefix := imageDirectoryPrefix
	if prefix == "" {
		homedir, _ := os.UserHomeDir()
		prefix = path.Join(homedir, constants.UserConfigDir, constants.UserImageCacheDir)
	}

	return path.Join(prefix, strings.ReplaceAll(imageName, "/", "_"))
}

// getSystemContext returns a SystemContext that makes sense
// for the system that the function is called on.  The specific
// behavior is undefined.  It should "do the right thing".
func getSystemContext(arch string) *types.SystemContext {
	rootPath := ""
	if runtime.GOOS == "darwin" {
		// If this fails, there are big problems
		homedir, _ := os.UserHomeDir()

		rootPath = path.Join(homedir, constants.UserConfigDir, constants.UserContainerConfigDir)
	}
	return &types.SystemContext{
		RootForImplicitAbsolutePaths: rootPath,
		OSChoice:                     "linux",
		ArchitectureChoice:           archMap[arch],
	}
}

// GetImageLayers fetches the BlobInfos for all the layers of the
// given ImageReference.
func GetImageLayers(imgRef types.ImageReference, arch string) ([]types.BlobInfo, error) {
	imgSrc, err := imgRef.NewImageSource(context.Background(), getSystemContext(arch))
	if err != nil {
		return nil, err
	}
	img, err := image.FromSource(context.Background(), getSystemContext(arch), imgSrc)
	if err != nil {
		return nil, err
	}

	return img.LayerInfos(), nil
}

// GetOrPull image ensures that a container image is on the filesystem
// of the local machine and returns a reference to that image.  If the
// image is already available locally, a reference to the existing image
// is returned.  If it is not present, then the image is pulled and then
// a reference is returned.
func GetOrPull(imageName string, arch string) (types.ImageReference, error) {
	srcRef, err := alltransports.ParseImageName(imageName)
	if err != nil {
		log.Debugf("Could not parse source image: %v", err)
		return nil, err
	}

	systemCtx := getSystemContext(arch)
	policy, err := signature.DefaultPolicy(systemCtx)
	if err != nil {
		log.Debugf("Could not get default signature policy: %v", err)
		return nil, err
	}

	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		log.Debugf("Could not get default policy context: %v", err)
		return nil, err
	}

	// The image is pulled to a local directory.  It's not the best
	// system, but it avoids the pain of having to set up a complete
	// image storage that lives aside the system default.
	dstDir := imageToDirectory(imageName)
	dstName := fmt.Sprintf("dir:%s", dstDir)
	err = os.MkdirAll(dstDir, 0700)
	if err != nil {
		log.Debugf("Could not create directory to contain image: %v", err)
		return nil, err
	}

	dstRef, err := alltransports.ParseImageName(dstName)
	if err != nil {
		log.Debugf("Could not parse destination reference: %v", err)
		return nil, err
	}

	// Check the pull directory for the manifest contents.
	// If its already there, then just return the reference.
	layers, err := GetImageLayers(srcRef, arch)
	if err != nil {
		log.Debugf("Could not make image from source reference: %v", err)
		return nil, err
	}

	imgExists := true
	for _, bi := range layers {
		blobPath := path.Join(dstDir, bi.Digest.Encoded())
		_, err := os.Stat(blobPath)
		if os.IsNotExist(err) {
			log.Debugf("Layer at %s does not exist", blobPath)
			imgExists = false
			break
		}
		if err != nil {
			log.Debugf("Error reading directory %s: %v", blobPath, err)
			return nil, err
		}
	}

	// Great! The image was found.  No need to re-pull, so
	// just hand the reference back
	if imgExists {
		log.Debugf("Found existing local image for: %s", imageName)
		return dstRef, nil
	}

	log.Debugf("Pulling image: %s", imageName)

	// The image was not found.  Pull it into the local storage
	// location, so it can be accessed later.
	copyOpts := &copy.Options{
		ReportWriter:       os.Stdout,
		SourceCtx:          getSystemContext(arch),
		ImageListSelection: copy.CopySystemImage,
	}
	_, err = copy.Image(
		context.Background(),
		policyCtx,
		dstRef,
		srcRef,
		copyOpts,
	)
	if err != nil {
		log.Debugf("Could not pull image: %v", err)
		return nil, err
	}

	return dstRef, err
}

// getLayerFile returns an open os.File for the file that backs an
// (image reference, layer id) pair.  This function assumes that the ImageReference
// uses the 'dir' transport mechanism.
func getLayerFile(imgRef types.ImageReference, layerId string) (*os.File, error) {
	fullPath := path.Join(imgRef.StringWithinTransport(), layerId)
	return os.Open(fullPath)
}

// GetTarFromLayerById opens a tar stream that reads the contents of the given layer for the given
// image.  The backing readers for the tar stream are a gzip stream and a file
// stream.  Both of these have to be closed.  The second return value is a
// function that closes each of these streams in the proper order.
func GetTarFromLayerById(imgRef types.ImageReference, layerId string) (*tar.Reader, func(), error) {
	layerFile, err := getLayerFile(imgRef, layerId)
	if err != nil {
		return nil, func() {}, err
	}

	// Layers are gzipped, so it is necessary to
	// unzip them.
	gzipReader, err := gzip.NewReader(layerFile)
	if err != nil {
		layerFile.Close()
		return nil, func() {}, err
	}

	// The unzipped file is a tar archive,
	// so set up a reader for that.
	tarReader := tar.NewReader(gzipReader)
	return tarReader,
		func() {
			gzipReader.Close()
			layerFile.Close()
		},
		nil
}

// AdvanceTarToPath searches a tar stream until it finds the header for
// the given file.  If the file is not found, the tar stream is advanced
// to the end, and it is no longer possible to read from it.
//
// For large archives, this can take a while.  The archive is usually
// read from disk, and is usually compressed.
func AdvanceTarToPath(tarReader *tar.Reader, filePath string) error {
	for {
		hdr, err := tarReader.Next()
		if err != nil {
			return err
		}

		if filePath == hdr.Name {
			return nil
		}
	}

	return fmt.Errorf("Reached unreachable code in AdvanceTarToPath")
}

// Login logs in to a container registry
func Login(registry string) error {
	opts := auth.LoginOptions{
		Stdin:              os.Stdin,
		Stdout:             os.Stdout,
		AcceptRepositories: true,
	}
	return auth.Login(context.Background(), getSystemContext(""), &opts, []string{registry})
}

// Copy moves an image from one place to another
func Copy(src string, dest string, arch string, imageSelection copy.ImageListSelection) error {
	srcRef, err := alltransports.ParseImageName(src)
	if err != nil {
		return err
	}

	destRef, err := alltransports.ParseImageName(dest)
	if err != nil {
		return err
	}

	systemCtx := getSystemContext(arch)
	policy, err := signature.DefaultPolicy(systemCtx)
	if err != nil {
		log.Debugf("Could not get default signature policy: %v", err)
		return err
	}

	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		log.Debugf("Could not get default policy context: %v", err)
		return err
	}

	copyOpts := &copy.Options{
		ReportWriter:       os.Stdout,
		SourceCtx:          systemCtx,
		ImageListSelection: imageSelection,
	}
	_, err = copy.Image(
		context.Background(),
		policyCtx,
		destRef,
		srcRef,
		copyOpts,
	)

	// If there is no error, then the copy is done.  If there
	// was an error, take a peek at it to see if it can
	// be resolved.
	if err == nil {
		return nil
	}

	// If the error is anything other than an authentication
	// error then bail
	if !strings.Contains(err.Error(), "denied") && !strings.Contains(err.Error(), "unauthorized") &&
		!strings.Contains(err.Error(), "403") && !strings.Contains(err.Error(), "not authorized") &&
		!strings.Contains(err.Error(), "authentication required") {
		return err
	}

	// If the error is with the source registry, do not attempt
	// to log in to the destination registry.
	if strings.Contains(err.Error(), "initializing source") {
		return err
	}

	// If the error is an auth problem but the target is not
	// a docker registry, then logging in has little hope of
	// resolving the problem.
	destName := destRef.DockerReference()
	srcName := srcRef.DockerReference()
	if destName == nil && srcName == nil {
		return err
	}

	//Only log in to the destination, assume the source does not require login
	if destName != nil {
		destDomain := reference.Domain(destName)
		log.Infof("Log in to %s", destDomain)
		err = Login(destDomain)
		if err != nil {
			return err
		}
	}

	_, err = copy.Image(
		context.Background(),
		policyCtx,
		destRef,
		srcRef,
		copyOpts,
	)

	return err
}

// ChangeImageDomain takes a image with a valid transport and domain and changes the domain
func ChangeImageDomain(imageString string, newDomain string) (types.ImageReference, error) {
	transport, afterTransport, found := strings.Cut(imageString, ":") // We may support non docker images in the future
	if !found {
		return nil, fmt.Errorf("could not remove the transport name from image %s", imageString)
	}
	imageRef, err := alltransports.ParseImageName(imageString) // The image name must have a transport to be valid
	if err != nil {
		return nil, fmt.Errorf("could not parse the image %s with error %v", imageString, err)
	}
	dockerRef := imageRef.DockerReference()
	if dockerRef == nil {
		return nil, fmt.Errorf("could not parse the image name, it is not a docker reference %s", imageString)
	}
	originalDomain := reference.Domain(imageRef.DockerReference()) // does not include the path i.e. /oracle/ui
	afterTransportBeforeDomain, afterDomain, found := strings.Cut(afterTransport, originalDomain)
	if !found {
		return nil, fmt.Errorf("could not remove the domain name %s from image name (transport removed): %s", originalDomain, afterTransport)
	}
	newDomain, _ = strings.CutSuffix(newDomain, "/")
	afterDomain, _ = strings.CutPrefix(afterDomain, "/")
	newURL := transport + ":" + afterTransportBeforeDomain + newDomain + "/" + afterDomain
	newImage, err := alltransports.ParseImageName(newURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse the image after changing the domain: %s, with error %v", newURL, err)
	}
	return newImage, nil
}

func WithoutTag(image string) (string, error) {
	fullImage, err := AddDefaultRegistry(image, "place.holder.com")
	if err != nil {
		return image, err
	}
	named, err := reference.ParseNamed(fullImage)
	if err != nil {
		return image, err
	}
	named = reference.TagNameOnly(named)
	tagged, ok := named.(reference.NamedTagged)
	if !ok {
		tmp := fmt.Sprintf("error removing image tag from string %s", image)
		return image, errors.New(tmp)
	}
	justImage, found := strings.CutSuffix(named.String(), ":"+tagged.Tag())
	if found {
		return justImage, nil
	}

	tmp := fmt.Sprintf("error removing image tag from string %s", image)
	return image, errors.New(tmp)
}
