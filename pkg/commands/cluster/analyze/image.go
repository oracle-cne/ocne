// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

func analyzeImages(p *analyzeParams) error {
	// Read the pod images into a map where the node name is the key and
	// tbe value is a PodmanImage list
	imageMap, err := readNodeSpecificJSONFiles[[]*PodmanImage](p, "podman-images.json")
	if err != nil {
		return err
	}

	// Read the pod images into a map where the node name is the key and
	// tbe value is a single PodmanInfo
	podmanInfoMap, err := readNodeSpecificJSONFiles[*PodmanInfo](p, "podman-info.json")
	if err != nil {
		return err
	}

	// Read the pod images into a map where the node name is the key and
	// tbe value is a PodmanImageDetail list
	imageDetailMap, err := readNodeSpecificJSONFiles[[]*PodmanImageDetail](p, "podman-inspect-all.json")
	if err != nil {
		return err
	}

	// Read the pod images into a map where the node name is the key and
	// tbe value is a PodmanImageDetail list
	imageDetailErrorsMap, err := readNodeSpecificTextFiles[[]*PodmanImageDetail](p, "podman-inspect-all-err.out")
	if err != nil {
		return err
	}

	//log.Infof("%d images found", len(imageMap))
	//log.Infof("%d image infos found", len(imageInfoMap))
	//log.Infof("%d image details found", len(imageDetailMap))

	data := PodmanImageData{
		PodmanInfoMap:       podmanInfoMap,
		ImageDetailMap:      imageDetailMap,
		ImageDetailErrorMap: imageDetailErrorsMap,
		ImageMap:            imageMap,
	}

	if p.verbose {
		return displayImageProblems(p.writer, &data)
	}
	return nil
}

func getImageErrors() {

}
