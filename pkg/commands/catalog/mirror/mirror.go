// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mirror

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	copyCommand "github.com/oracle-cne/ocne/pkg/commands/catalog/copy"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/helm"
	imageUtil "github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1Apps "k8s.io/api/apps/v1"
	v1Batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

var appsWithRequiredValues = map[string]map[string]string{
	"oci-capi": {
		"authConfig.fingerprint":          "placeholder-fingerprint",
		"authConfig.key":                  "placeholder-key",
		"authConfig.passphrase":           "placeholder-password",
		"authConfig.region":               "us-ashburn-1",
		"authConfig.tenancy":              "placeholder-tenancy",
		"authConfig.useInstancePrincipal": "placeholder-principal",
		"authConfig.user":                 "ocne-user",
	},
}

type Options struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// CatalogName is the name of the catalog to mirror
	CatalogName string

	//  DestinationURI is the URI of the destination repo
	DestinationURI string

	// Push is true if images are to be pushed to the destinationURI
	Push bool

	// ConfigURI is the URI of the OCNE config file, only the applications in this file are mirrored
	ConfigURI string

	// Config is the OCNE configuration
	Config *types.Config

	// ClusterConfig is the cluster configuration
	ClusterConfig *types.ClusterConfig

	// DefaultRegistry is the registry to add to images without a domain. Stores the --source argument.
	DefaultRegistry string
}

const extraImagesLabel string = "extra-image"

// Mirror extracts images from charts in an app catalog and optionally pushes them to a destination.
func Mirror(options Options) error {
	var toIterate map[string][]catalog.ChartMeta
	found, err := performSearch(".*", options.CatalogName, options.KubeConfigPath)
	if err != nil {
		return err
	}
	toIterate = found.ChartEntries // indexed by app name, not release name
	toIterate = filter(toIterate, options.ClusterConfig.Applications, options.CatalogName)
	toRender, err := addOverrides(toIterate, options.ClusterConfig.Applications)
	if err != nil {
		return err
	}
	allVersions, err := versions.GetKubernetesVersions(options.ClusterConfig.KubeVersion)
	if err != nil {
		return err
	}
	k8sVersion, err := semver.NewVersion(allVersions.Kubernetes)
	if err != nil {
		return err
	}
	images, err := extractImages(toRender, found, options.KubeConfigPath, k8sVersion)
	if err != nil {
		return err
	}
	images = imageUtil.AddDefaultRegistries(images, options.DefaultRegistry)
	images = removeDuplicates(images)

	if !options.Push {
		for _, image := range images {
			fmt.Printf("%s\n", image)
		}
	}

	if options.Push && options.DestinationURI == "" {
		return errors.New("Please provide a destination URI")
	}

	if options.Push {
		co := catalog.CopyOptions{
			KubeConfigPath: options.KubeConfigPath,
			Destination:    options.DestinationURI,
			Images:         images,
		}
		err = copyCommand.Copy(co)
	}

	// TODO: Other future functionality, save images to a local cache and tag them
	return err
}

func removeDuplicates(arr []string) []string {
	helper := map[string]bool{}
	for _, i := range arr {
		helper[i] = true
	}
	newArr := make([]string, 0)
	for key := range helper {
		newArr = append(newArr, key)
	}
	return newArr
}

// addOverrides adds overrides to all applications in the apps list, including any hardcoded overrides.
func addOverrides(toIterate map[string][]catalog.ChartMeta, apps []types.Application) (map[string][]catalog.ChartWithOverrides, error) {
	toReturn := make(map[string][]catalog.ChartWithOverrides)
	if len(apps) == 0 {
		for key, app := range toIterate {
			var val map[string]string = nil
			//see if we have hardcoded overrides, this avoids issues with helm install
			if mapOfValues, found := appsWithRequiredValues[key]; found {
				val = mapOfValues
			}
			for _, chart := range app {
				temp, err := buildOverride(chart, nil, "", val)
				if err != nil {
					log.Errorf("Error adding overrides to chart %s, error: %s", key, err.Error())
				}
				toReturn[key] = append(toReturn[key], temp)
			}
		}
		return toReturn, nil
	}
	for _, app := range apps {
		var val map[string]string = nil
		if mapOfValues, found := appsWithRequiredValues[app.Name]; found {
			val = mapOfValues // can potentially overwrite user overrides, but should be a nonissue
		}
		for _, chart := range toIterate[app.Name] {
			temp, err := buildOverride(chart, app.Config, app.ConfigFrom, val)
			if err != nil {
				log.Errorf("Error adding overrides to chart %s, error: %s", app.Name, err.Error())
			}
			toReturn[app.Name] = append(toReturn[app.Name], temp)
		}
	}
	return toReturn, nil
}

// buildOverride adds overrides to one application. confInterface is a yaml representing a values.yaml file. configFrom is a URI to a values.yaml file.
func buildOverride(chart catalog.ChartMeta, confInterface interface{}, configFrom string, setOverridesMap map[string]string) (catalog.ChartWithOverrides, error) {
	configVal := ""
	setOverridesString := ""
	if configFrom != "" {
		configFrom = strings.TrimRight(configFrom, "\n")
	}
	if confInterface != nil {
		//should be some explicit setting of values in yaml format only
		yamlValues, err := yaml.Marshal(confInterface)
		if err != nil {
			return catalog.ChartWithOverrides{
				Chart: chart,
				Overrides: []helm.HelmOverrides{
					{
						FileOverride: configFrom,
					},
				},
			}, err
		}
		configVal = strings.TrimRight(string(yamlValues), "\n")
	}
	if setOverridesMap != nil {
		for key, val := range setOverridesMap {
			setOverridesString += key + "=" + val + ","
		}
		setOverridesString = strings.TrimSuffix(setOverridesString, ",")
	}
	toReturn := catalog.ChartWithOverrides{
		Chart: chart,
		Overrides: []helm.HelmOverrides{
			{
				FileOverride: configFrom,
			},
			{
				LiteralOverride: configVal,
			},
			{
				SetOverrides: setOverridesString,
			},
		},
	}
	return toReturn, nil
}

// filter takes a map of chart entries found in a catalog and applications specified by the user in the cluster config.
//  1. If no applications are specified in the cluster config
//     a. Return all chart entries
//  2. Else
//     b. For each app
//     i. If the user specified a version for the app, add that to the return value
//     ii. Else grab the highest version of the app in the catalog
func filter(toIterate map[string][]catalog.ChartMeta, search []types.Application, catalogName string) map[string][]catalog.ChartMeta {
	toReturn := make(map[string][]catalog.ChartMeta)
	if len(search) == 0 {
		return toIterate
	}
	for _, app := range search {
		if strings.TrimSpace(app.Catalog) != strings.TrimSpace(catalogName) {
			log.Debugf("Skipping helm Chart %s, it is in a different catalog: %s", app.Name, app.Catalog)
			continue
		}
		if _, ok := toIterate[app.Name]; !ok {
			log.Debugf("Skipping helm Chart %s, not found in repo %s", app.Name, app.Catalog)
			continue
		}
		var toAdd catalog.ChartMeta
		if app.Version == "" {
			//grab the latest version
			foundMax := false
			var maxSemver *semver.Version
			var maxChart catalog.ChartMeta
			for _, chart := range toIterate[app.Name] {
				chartSemver, err := semver.NewVersion(chart.Version)
				if err != nil {
					log.Debugf("Incorrect version number %s for chart %s. Not considering this version", app.Version, app.Name)
					continue
				}
				if !foundMax || chartSemver.GreaterThan(maxSemver) {
					maxChart = chart
					maxSemver, _ = semver.NewVersion(maxChart.Version)
					foundMax = true
				}
			}
			if !foundMax {
				log.Errorf("No correct semver format versions found for chart %s, defaulting to version %s for this chart", app.Name, toIterate[app.Name][0].Version)
				toAdd = toIterate[app.Name][0]
			} else {
				log.Debugf("Grabbed the latest version %s of chart %s", maxChart.Version, maxChart.Name)
				toAdd = maxChart
			}
		} else {
			//grab app.Version
			for _, chart := range toIterate[app.Name] {
				if chart.Version == app.Version {
					toAdd = chart
					break
				}
			}
			if toAdd.Version != app.Version {
				log.Errorf("No chart of name %s with version %s found in the app catalog. To grab the latest version of the chart, remove the application version key from the cluster config. Defaulting to version %s for this chart", app.Name, app.Version, toIterate[app.Name][0].Version)
				toAdd = toIterate[app.Name][0]
			}
		}
		//Accounts for the case where a user has multiple versions of the same app
		toReturn[app.Name] = append(toReturn[app.Name], toAdd)
	}
	return toReturn
}

func performSearch(regex string, catalogName string, kPath string) (*catalog.Catalog, error) {
	so := catalog.SearchOptions{
		KubeConfigPath: kPath,
		CatalogName:    catalogName,
		Pattern:        regex,
	}
	c, err := search.Search(so)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// extractImages takes a map of chart metadata with overrides to grab the raw charts from a catalog.
// It applies the overrides to the raw charts in a helm template command and parses the resulting yaml k8s objects for images.
func extractImages(meta map[string][]catalog.ChartWithOverrides, chartCatalog *catalog.Catalog, kc string, k8sVersion *semver.Version) ([]string, error) {
	// Ignore errors getting the client.  It probably already succeeded due
	// to the catalog search.  If not, it will probably fail later when
	// attempting to render the charts.  The error is being ignored to
	// protect against cases where no kubeconfig is available but the chart
	// is.  For example, when using the embedded catalog.
	kubeInfo, _ := client.CreateKubeInfo(kc)
	images := make([]string, 0)

	for name, chartMetaList := range meta {
		for _, chartWithOverrides := range chartMetaList {
			chart, err := chartCatalog.Connection.GetChart(name, chartWithOverrides.Chart.Version)
			if err != nil {
				log.Errorf("Skipping helm Chart %s, error getting chart from catalog", name)
				log.Error(err.Error())
				continue
			}
			in := bytes.NewReader(chart)
			archive, err := loader.LoadArchive(in)
			if err != nil {
				log.Errorf("Skipping helm Chart %s, error reading chart bytes", name)
				log.Error(err.Error())
				continue
			}
			manifest, err := helm.Template(kubeInfo, archive, chartWithOverrides.Overrides, k8sVersion)
			if err != nil {
				log.Errorf("Skipping helm Chart %s, error rendering chart", name)
				log.Error(err.Error())
				continue
			}
			extraImages := grabExtraImages(manifest)
			listOfYamlStrings := splitYamls(manifest) //split yamls so that unmarshall errors are not impossible to read
			k8sObjects := unmarshallObjects(listOfYamlStrings)
			grabbed := imagesFromObjects(k8sObjects)
			if len(grabbed) > 0 {
				images = append(images, grabbed...)
			}
			if len(extraImages) > 0 {
				images = append(images, extraImages...)
			}
		}
	}
	if len(images) == 0 {
		return images, errors.New("no container images found")
	}
	return images, nil
}

// imagesFromObjects grabs the image field from all k8s objects that have it
func imagesFromObjects(objects []unstructured.Unstructured) []string {
	toReturn := make([]string, 0)

	fromPodSpec := func(spec v1.PodSpec) []string {
		for _, container := range spec.Containers {
			toReturn = append(toReturn, container.Image)
		}
		for _, initContainer := range spec.InitContainers {
			toReturn = append(toReturn, initContainer.Image)
		}
		for _, ephemeralContainer := range spec.EphemeralContainers {
			toReturn = append(toReturn, ephemeralContainer.Image)
		}
		return toReturn
	}

	for _, object := range objects {
		objectType := strings.ToLower(object.GetKind())
		content := object.UnstructuredContent()
		converter := runtime.DefaultUnstructuredConverter
		//grab images from all workload resources
		switch objectType {
		case "pod":
			var pod v1.Pod
			err := converter.FromUnstructured(content, &pod)
			if err != nil {
				log.Errorf("Could not parse images from a pod %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(pod.Spec)
			toReturn = append(toReturn, imgs...)
		case "podtemplate":
			var podTemplate v1.PodTemplate
			err := converter.FromUnstructured(content, &podTemplate)
			if err != nil {
				log.Errorf("Could not parse images from a pod template %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(podTemplate.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "deployment":
			var deployment v1Apps.Deployment
			err := converter.FromUnstructured(content, &deployment)
			if err != nil {
				log.Errorf("Could not parse images from a deployment %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(deployment.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "replicationcontroller":
			var replicaController v1.ReplicationController
			err := converter.FromUnstructured(content, &replicaController)
			if err != nil {
				log.Errorf("Could not parse images from a replication controller %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(replicaController.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "replicaset":
			var replicaSet v1Apps.ReplicaSet
			err := converter.FromUnstructured(content, &replicaSet)
			if err != nil {
				log.Errorf("Could not parse images from a replica set %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(replicaSet.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "statefulset":
			var statefulSet v1Apps.StatefulSet
			err := converter.FromUnstructured(content, &statefulSet)
			if err != nil {
				log.Errorf("Could not parse images from a stateful set %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(statefulSet.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "daemonset":
			var daemonSet v1Apps.DaemonSet
			err := converter.FromUnstructured(content, &daemonSet)
			if err != nil {
				log.Errorf("Could not parse images from a daemon set %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(daemonSet.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "job":
			var job v1Batch.Job
			err := converter.FromUnstructured(content, &job)
			if err != nil {
				log.Errorf("Could not parse images from a job %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(job.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		case "cronjob":
			var cronJob v1Batch.CronJob
			err := converter.FromUnstructured(content, &cronJob)
			if err != nil {
				log.Errorf("Could not parse images from a cron job %s, error:", object.GetName())
				log.Error(err.Error())
				continue
			}
			imgs := fromPodSpec(cronJob.Spec.JobTemplate.Spec.Template.Spec)
			toReturn = append(toReturn, imgs...)
		default:
			log.Debugf("Skipping k8s object kind %s, name %s", object.GetKind(), object.GetName())
		}
	}
	return toReturn
}

// grabExtraImages returns non-whitespace text that follows the extraImagesLabel in any string
func grabExtraImages(manifest string) []string {
	re, err := regexp.Compile(`(#\s*` + extraImagesLabel + `:\s*)(\S*)`)
	if err != nil {
		return []string{}
	}

	found := re.FindAllStringSubmatch(manifest, -1)
	if len(found) == 0 {
		log.Debug("No extra images label found in charts")
	}
	toReturn := make([]string, len(found))
	for i, find := range found {
		toReturn[i] = find[2]
		log.Debugf("Found extra container image %s in charts", find[2])
	}
	return toReturn
}

// splitYamls uses the yaml end of document delimiter. It does not support nested delimiters and delimiters preceded by whitespace.
func splitYamls(manifest string) []string {
	temp := strings.Split(manifest, "\n---")
	if len(temp) > 0 {
		temp[0], _ = strings.CutPrefix(temp[0], "---\n")
	}
	return temp
}

func unmarshallObjects(listOfYamls []string) []unstructured.Unstructured {
	toReturn := make([]unstructured.Unstructured, 0)
	for _, yamlString := range listOfYamls {
		yamlString = strings.TrimSpace(yamlString)
		stringReader := strings.NewReader(yamlString)
		bufReader := bufio.NewReader(stringReader)
		temp, err := k8s.Unmarshall(bufReader)
		if (err != nil) || (len(temp) <= 0) {
			log.Debugf("Skipping a yaml string, could not unmarshall: \"%s\"", yamlString)
			if err != nil {
				log.Debug(err.Error())
			}
			continue
		}
		toReturn = append(toReturn, temp...)
	}
	return toReturn
}
