// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

const ReleaseNotFound = "NotFound"
const ReleaseStatusDeployed = "deployed"
const ReleaseStatusFailed = "failed"

// ChartStatusFnType - Package-level var and functions to allow overriding GetChartStatus for unit test purposes
type ChartStatusFnType func(releaseName string, namespace string) (string, error)

// HelmOverrides contains all the overrides that gets passed to the helm cli runner
type HelmOverrides struct {
	LiteralOverride    string // literal yaml
	SetOverrides       string // for --set
	SetStringOverrides string // for --set-string
	SetFileOverrides   string // for --set-file
	FileOverride       string // for -f
}

type ActionConfigFnType func(kubeInfo *client.KubeInfo, namespace string) (*action.Configuration, error)

var actionConfigFn ActionConfigFnType = getActionConfig

func SetActionConfigFunction(f ActionConfigFnType) {
	actionConfigFn = f
}

// SetDefaultActionConfigFunction Resets the action config function
func SetDefaultActionConfigFunction() {
	actionConfigFn = getActionConfig
}

type LoadChartFnType func(chartDir string) (*chart.Chart, error)

var loadChartFn LoadChartFnType = loadChart

type RestGetter struct {
	kubeClient *rest.Config
}

func SetLoadChartFunction(f LoadChartFnType) {
	loadChartFn = f
}

func SetDefaultLoadChartFunction() {
	loadChartFn = loadChart
}

// GetRelease will run 'helm get all' command and return the output from the command.
func GetRelease(kubeInfo *client.KubeInfo, releaseName string, namespace string) (*release.Release, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGet(actionConfig)
	return client.Run(releaseName)
}

// GetValuesMap will run 'helm get values' command and return the output from the command.
func GetValuesMap(kubeInfo *client.KubeInfo, releaseName string, namespace string) (map[string]interface{}, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGetValues(actionConfig)
	vals, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// GetMetadata will run 'helm get metadata' command and return the output from the command.
func GetMetadata(kubeInfo *client.KubeInfo, releaseName string, namespace string) (*action.Metadata, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGetMetadata(actionConfig)
	vals, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// GetAllValuesMap will run 'helm get values -a' command and return the output from the command.
func GetAllValuesMap(kubeInfo *client.KubeInfo, releaseName string, namespace string) (map[string]interface{}, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGetValues(actionConfig)
	client.AllValues = true
	vals, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// GetAllValues will run 'helm get values -a' command and return the output from the command.
func GetAllValues(kubeInfo *client.KubeInfo, releaseName string, namespace string) ([]byte, error) {
	vals, err := GetAllValuesMap(kubeInfo, releaseName, namespace)
	if err != nil {
		return nil, err
	}

	yamlValues, err := yaml.Marshal(vals)
	if err != nil {
		return nil, err
	}
	return yamlValues, nil
}

// GetValues will run 'helm get values' command and return the output from the command.
func GetValues(kubeInfo *client.KubeInfo, releaseName string, namespace string) ([]byte, error) {
	vals, err := GetValuesMap(kubeInfo, releaseName, namespace)
	if err != nil {
		return nil, err
	}

	yamlValues, err := yaml.Marshal(vals)
	if err != nil {
		return nil, err
	}
	return yamlValues, nil
}

// UpgradeChartFromArchive installs a release into a cluster or upgrades an existing one.  The
// reader must be a compressed tar archive.
func UpgradeChartFromArchive(kubeInfo *client.KubeInfo, releaseName string, namespace string, createNamespace bool, archive io.Reader, wait bool, dryRun bool, overrides []HelmOverrides, resetValues bool) (*release.Release, error) {
	theChart, err := loader.LoadArchive(archive)
	if err != nil {
		log.Errorf("Error loading archive: %v", err)
		return nil, err
	}

	return UpgradeChart(kubeInfo, releaseName, namespace, createNamespace, theChart, wait, dryRun, overrides, resetValues)
}

// UpgradeChart installs a release into a cluster or upgrades an existing one.
func UpgradeChart(kubeInfo *client.KubeInfo, releaseName string, namespace string, createNamespace bool, theChart *chart.Chart, wait bool, dryRun bool, overrides []HelmOverrides, resetValues bool) (*release.Release, error) {
	var err error
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}
	settings := cli.New()
	settings.KubeConfig = kubeInfo.KubeconfigPath
	settings.KubeTLSServerName = kubeInfo.KubeApiServerIP
	settings.SetNamespace(namespace)

	p := getter.All(settings)
	p = append(p, NewGitProvider())
	vals, err := MergeValues(overrides, p)
	if err != nil {
		return nil, err
	}

	installed, err := IsReleaseInstalled(kubeInfo, releaseName, namespace)
	if err != nil {
		return nil, err
	}

	var rel *release.Release
	if installed {
		// upgrade it
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.DryRun = dryRun
		client.Wait = wait
		client.ResetValues = resetValues

		// Reuse the original set of input values as the base set of helm overrides
		helmValues := map[string]interface{}{}
		if !resetValues {
			helmValuesTemp, err := GetValuesMap(kubeInfo, releaseName, namespace)
			if err != nil {
				return nil, err
			}
			maps.Copy(helmValues, helmValuesTemp)
		}

		// Append the new override values
		for k, v := range vals {
			helmValues[k] = v
		}

		rel, err = client.Run(releaseName, theChart, helmValues)
		if err != nil {
			fmt.Errorf("Failed running Helm command for release %s: %v",
				releaseName, err)
			return nil, err
		}
	} else {
		client := action.NewInstall(actionConfig)
		client.Namespace = namespace
		client.ReleaseName = releaseName
		client.DryRun = dryRun
		client.Replace = true
		client.Wait = wait
		client.CreateNamespace = createNamespace

		rel, err = client.Run(theChart, vals)
		if err != nil {
			fmt.Errorf("Failed running Helm command for release %s: %v",
				releaseName, err)
			return nil, err
		}
	}

	return rel, nil
}

// Template templates a chart using the provided overrides and returns the generated yamls as a string
func Template(kubeInfo *client.KubeInfo, theChart *chart.Chart, overrides []HelmOverrides, k8sVersion *semver.Version) (string, error) {
	actionConfig, err := actionConfigFn(kubeInfo, "")
	if err != nil {
		return "", err
	}
	settings := cli.New()
	settings.SetNamespace("")
	if kubeInfo != nil {
		settings.KubeConfig = kubeInfo.KubeconfigPath
		settings.KubeTLSServerName = kubeInfo.KubeApiServerIP
	}
	installClient := action.NewInstall(actionConfig)
	installClient.Namespace = "dummyNamespace"
	installClient.ReleaseName = theChart.Name()
	installClient.UseReleaseName = false
	installClient.GenerateName = true
	installClient.DryRun = true
	installClient.ClientOnly = true
	installClient.SkipCRDs = false
	installClient.IncludeCRDs = true
	installClient.Wait = false
	//installClient.CreateNamespace = true
	installClient.KubeVersion = &chartutil.KubeVersion{
		Version: k8sVersion.String(),
		Major:   strconv.FormatUint(k8sVersion.Major(), 10),
		Minor:   strconv.FormatUint(k8sVersion.Minor(), 10),
	}

	p := getter.All(settings)
	vals, err := MergeValues(overrides, p)
	if err != nil {
		return "", err
	}

	rel, err := installClient.Run(theChart, vals)
	if err != nil {
		return "", err
	}

	out := rel.Manifest
	for _, h := range rel.Hooks {
		out = fmt.Sprintf("%s\n---\n%s", out, h.Manifest)
	}

	return out, nil
}

// Uninstall will uninstall the helmRelease in the specified namespace using helm uninstall
func Uninstall(kubeInfo *client.KubeInfo, releaseName string, namespace string, dryRun bool) (err error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return err
	}

	client := action.NewUninstall(actionConfig)
	client.DryRun = dryRun

	_, err = client.Run(releaseName)
	if err != nil {
		return fmt.Errorf("Error uninstalling release %s: %s", releaseName, err.Error())
	}

	return nil
}

// maskSensitiveData replaces sensitive data in a string with mask characters.
func maskSensitiveData(str string) string {
	const maskString = "*****"
	re := regexp.MustCompile(`[Pp]assword=(.+?)(?:,|\z)`)

	matches := re.FindAllStringSubmatch(str, -1)
	for _, match := range matches {
		if len(match) == 2 {
			str = strings.Replace(str, match[1], maskString, 1)
		}
	}

	return str
}

// IsReleaseFailed Returns true if the chart helmRelease state is marked 'failed'
func IsReleaseFailed(kubeInfo *client.KubeInfo, releaseName string, namespace string) (bool, error) {
	releaseStatus, err := getReleaseState(kubeInfo, releaseName, namespace)
	if err != nil {
		err := fmt.Errorf("Getting status for chart %s/%s failed", namespace, releaseName)
		return false, err
	}
	return releaseStatus == ReleaseStatusFailed, nil
}

// IsReleaseDeployed returns true if the helmRelease is deployed
func IsReleaseDeployed(kubeInfo *client.KubeInfo, releaseName string, namespace string) (found bool, err error) {
	releaseStatus, err := GetHelmReleaseStatus(kubeInfo, releaseName, namespace)
	if err != nil {
		err := fmt.Errorf("Getting status for chart %s/%s failed with error: %v\n", namespace, releaseName, err)
		return false, err
	}
	switch releaseStatus {
	case ReleaseStatusDeployed:
		return true, nil
	}
	return false, nil
}

// GetReleaseStatus returns the helmRelease status
func GetReleaseStatus(kubeInfo *client.KubeInfo, releaseName string, namespace string) (status string, err error) {
	releaseStatus, err := GetHelmReleaseStatus(kubeInfo, releaseName, namespace)
	if err != nil {
		return "", fmt.Errorf("Failed getting status for chart %s/%s with stderr: %v\n", namespace, releaseName, err)
	}

	return releaseStatus, nil
}

// IsReleaseInstalled returns true if the helmRelease is installed
func IsReleaseInstalled(kubeInfo *client.KubeInfo, releaseName string, namespace string) (found bool, err error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return false, err
	}

	client := action.NewStatus(actionConfig)
	helmRelease, err := client.Run(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return release.StatusDeployed == helmRelease.Info.Status, nil
}

// GetHelmReleaseStatus extracts the Helm deployment status of the specified chart from the JSON output as a string
func GetHelmReleaseStatus(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return "", err
	}

	client := action.NewStatus(actionConfig)
	helmRelease, err := client.Run(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ReleaseNotFound, nil
		}
		return "", err
	}

	return helmRelease.Info.Status.String(), nil
}

// getReleaseState extracts the helmRelease state from an "ls -o json" command for a specific helmRelease/namespace
func getReleaseState(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	releases, err := GetReleases(kubeInfo, namespace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ReleaseNotFound, nil
		}
		return "", err
	}

	status := ""
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			status = info.Info.Status.String()
			break
		}
	}
	return strings.TrimSpace(status), nil
}

// GetReleaseAppVersion - public function to execute releaseAppVersionFn
func GetReleaseAppVersion(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	return getReleaseAppVersion(kubeInfo, releaseName, namespace)
}

// GetReleaseStringValues - Returns a subset of Helm helmRelease values as a map of strings
func GetReleaseStringValues(kubeInfo *client.KubeInfo, valueKeys []string, releaseName string, namespace string) (map[string]string, error) {
	values, err := GetReleaseValues(kubeInfo, valueKeys, releaseName, namespace)
	if err != nil {
		return map[string]string{}, err
	}
	returnVals := map[string]string{}
	for key, val := range values {
		returnVals[key] = fmt.Sprintf("%v", val)
	}
	return returnVals, err
}

// GetReleaseValues - Returns a subset of Helm helmRelease values as a map of objects
func GetReleaseValues(kubeInfo *client.KubeInfo, valueKeys []string, releaseName string, namespace string) (map[string]interface{}, error) {
	isDeployed, err := IsReleaseDeployed(kubeInfo, releaseName, namespace)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var values = map[string]interface{}{}
	if isDeployed {
		valuesMap, err := GetValuesMap(kubeInfo, releaseName, namespace)
		if err != nil {
			return map[string]interface{}{}, err
		}
		for _, valueKey := range valueKeys {
			if mapVal, ok := valuesMap[valueKey]; ok {
				values[valueKey] = mapVal
			}
		}
	}
	return values, nil
}

// getReleaseAppVersion extracts the helmRelease app_version from a "ls -o json" command for a specific helmRelease/namespace
func getReleaseAppVersion(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	releases, err := GetReleases(kubeInfo, namespace)
	if err != nil {
		if err.Error() == ReleaseNotFound {
			return ReleaseNotFound, nil
		}
		return "", err
	}

	var status string
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			status = info.Chart.AppVersion()
			break
		}
	}
	return strings.TrimSpace(status), nil
}

func getReleases(kubeInfo *client.KubeInfo, namespace string, allNamespaces bool) ([]*release.Release, error) {
	actionConfig, err := actionConfigFn(kubeInfo, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = allNamespaces
	client.All = true
	client.StateMask = action.ListAll

	releasesIface, _, err := util.ExponentialRetry(func(arg interface{}) (interface{}, bool, error) {
		client, _ := arg.(*action.List)
		releases, err := client.Run()
		return releases, false, err
	}, client)

	releases, _ := releasesIface.([]*release.Release)

	return releases, nil
}

func GetReleases(kubeInfo *client.KubeInfo, namespace string) ([]*release.Release, error) {
	return getReleases(kubeInfo, namespace, false)
}

// GetReleasesAllNamespaces gets the release information over all namespaces
// In the helm source code, the namespace for the envSettings and the configuration seem to be set to ""
// when the --all-namespaces flag is used, so I am following that pattern
func GetReleasesAllNamespaces(kubeInfo *client.KubeInfo) ([]*release.Release, error) {
	return getReleases(kubeInfo, "", true)
}

func debugLog(format string, v ...interface{}) {
}

func getActionConfig(kubeInfo *client.KubeInfo, namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	config := &genericclioptions.ConfigFlags{}
	if kubeInfo != nil {
		config.KubeConfig = &kubeInfo.KubeconfigPath
		config.BearerToken = &kubeInfo.RestConfig.BearerToken
		config.TLSServerName = &kubeInfo.KubeApiServerIP
		config.Namespace = &namespace

	}

	if err := actionConfig.Init(config, namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

func loadChart(chartDir string) (*chart.Chart, error) {
	return loader.Load(chartDir)
}

// readFile load a file using a URI scheme provider
func readFile(filePath string, p getter.Providers) ([]byte, error) {
	if strings.TrimSpace(filePath) == "-" {
		return io.ReadAll(os.Stdin)
	}
	u, err := url.Parse(filePath)
	if err != nil {
		return nil, err
	}

	g, err := p.ByScheme(u.Scheme)
	if err != nil {
		return os.ReadFile(filePath)
	}
	data, err := g.Get(filePath, getter.WithURL(filePath))
	if err != nil {
		return nil, err
	}
	return data.Bytes(), err
}

// MergeValues merges values from the specified overrides
func MergeValues(overrides []HelmOverrides, p getter.Providers) (map[string]interface{}, error) {
	base := map[string]interface{}{}
	var err error

	// User specified a values files via -f/--values
	for _, override := range overrides {
		if len(override.LiteralOverride) > 0 {
			currentMap := map[string]interface{}{}
			if err := yaml.Unmarshal([]byte(override.LiteralOverride), &currentMap); err != nil {
				return nil, err
			}
			err = MergeMaps(base, currentMap)
			if err != nil {
				return nil, err
			}
		}
		if len(override.FileOverride) > 0 {
			currentMap := map[string]interface{}{}

			var bytes []byte
			bytes, err = readFile(override.FileOverride, p)
			if err != nil {
				return nil, err
			}

			if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
				return nil, err
			}
			// Merge with the previous map
			err = MergeMaps(base, currentMap)
			if err != nil {
				return nil, err
			}
		}

		// User specified a value via --set
		if len(override.SetOverrides) > 0 {
			if err := strvals.ParseInto(override.SetOverrides, base); err != nil {
				return nil, err
			}
		}

		// User specified a value via --set-string
		if len(override.SetStringOverrides) > 0 {
			if err := strvals.ParseIntoString(override.SetStringOverrides, base); err != nil {
				return nil, err
			}
		}

		// User specified a value via --set-file
		if len(override.SetFileOverrides) > 0 {
			reader := func(rs []rune) (interface{}, error) {
				bytes, err := readFile(string(rs), p)
				if err != nil {
					return nil, err
				}
				return string(bytes), err
			}
			if err := strvals.ParseIntoFile(override.SetFileOverrides, base, reader); err != nil {
				return nil, err
			}
		}
	}

	return base, nil
}

// MergeMaps merges two maps where mOverlay overrides mBase
func MergeMaps(mBase map[string]interface{}, mOverlay map[string]interface{}) error {
	for k, vOverlay := range mOverlay {
		vBase, ok := mBase[k]
		recursed := false
		if ok {
			// Both mBase and mOverlay have this key. If these are nested maps merge them
			switch tBase := vBase.(type) {
			case map[string]interface{}:
				switch tOverlay := vOverlay.(type) {
				case map[string]interface{}:
					MergeMaps(tBase, tOverlay)
					recursed = true
				}
			}
		}
		// Both values were not maps, put overlay entry into the base map
		// This might be a new base entry or replaced by the mOverlay value
		if !recursed {
			mBase[k] = vOverlay
		}
	}
	return nil
}

// GetShowValues returns the values.yaml file of a particular chart
func GetShowValues(archive io.Reader) ([]byte, error) {
	var bytesToReturn []byte
	theChart, err := loader.LoadArchive(archive)
	if err != nil {
		log.Errorf("Error loading archive: %v", err)
		return bytesToReturn, err
	}
	files := theChart.Raw
	for _, file := range files {
		if file.Name == chartutil.ValuesfileName {
			bytesToReturn = file.Data
			return bytesToReturn, nil
		}
	}
	return bytesToReturn, fmt.Errorf("Values.yaml was not found ")

}
