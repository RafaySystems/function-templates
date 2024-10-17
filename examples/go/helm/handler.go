package function

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"helm.sh/helm/v3/pkg/action"
	h3a "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	h3cli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	h3r "helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Config struct {
	Action       string         `json:"action"`
	Kubeconfig   string         `json:"kubeconfig"`
	Namespace    string         `json:"namespace"`
	Release      string         `json:"release"`
	RepoName     string         `json:"repo_name"`
	RepoURL      string         `json:"repo_url"`
	ChartVersion string         `json:"chart_version"`
	HelmValues   map[string]any `json:"helm_values"`
	tmpDir       string
	settings     *h3cli.EnvSettings
	cfgFlags     *genericclioptions.ConfigFlags
	actionConfig *h3a.Configuration
	checkStatus  string
}

func Handle(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
	logger.Info("received request")

	// Parse the request
	cfg, err := parseConfig(req)
	if err != nil {
		return nil, err
	}

	// create tmp dir
	cfg.tmpDir, err = os.MkdirTemp(os.TempDir(), "helm-*")
	if err != nil {
		return nil, sdk.NewErrFailed("failed to create temp dir")
	}
	defer os.RemoveAll(cfg.tmpDir)

	// create kubeconfig file
	kubeconfigFile, err := createKubeconfigFile(cfg)
	if err != nil {
		return nil, sdk.NewErrFailed("failed to create kubeconfig file")
	}

	// Set up Helm configuration
	settings := h3cli.New()
	settings.PluginsDirectory = path.Join(cfg.tmpDir, "plugins")
	settings.RepositoryConfig = path.Join(cfg.tmpDir, "repositories.yaml")
	settings.RepositoryCache = path.Join(cfg.tmpDir, "repository")
	settings.RegistryConfig = path.Join(cfg.tmpDir, "registry/config.json")

	cfgFlags := genericclioptions.NewConfigFlags(true)
	cfgFlags.KubeConfig = &kubeconfigFile
	cfgFlags.CacheDir = &cfg.tmpDir

	actionConfig := new(h3a.Configuration)
	actionConfig.Init(cfgFlags, cfg.Namespace, "secret", logger.Debug)

	cfg.settings = settings
	cfg.cfgFlags = cfgFlags
	cfg.actionConfig = actionConfig

	// Perform the action
	switch cfg.Action {
	case "deploy":
		return deploy(logger, cfg)
	case "destroy":
		return destroy(logger, cfg)
	default:
		return nil, sdk.NewErrFailed(fmt.Sprintf("invalid action - %s", cfg.Action))
	}
}

func parseConfig(req sdk.Request) (*Config, error) {
	cfg := &Config{}

	action, err := req.GetString("action")
	if err != nil || action == "" {
		action = "deploy"
	}
	cfg.Action = action

	namespace, err := req.GetString("namespace")
	if err != nil || namespace == "" {
		return nil, sdk.NewErrFailed("missing namespace")
	}
	cfg.Namespace = namespace

	kubeconfig, err := req.GetString("kubeconfig")
	if err != nil || kubeconfig == "" {
		return nil, sdk.NewErrFailed("missing kubeconfig")
	}
	cfg.Kubeconfig = kubeconfig

	release, err := req.GetString("release")
	if err != nil || release == "" {
		return nil, sdk.NewErrFailed("missing release name")
	}
	cfg.Release = release

	repoName, _ := req.GetString("repo_name")
	cfg.RepoName = repoName

	repoUrl, err := req.GetString("repo_url")
	if err != nil || repoUrl == "" {
		return nil, sdk.NewErrFailed("missing repo url")
	}
	cfg.RepoURL = repoUrl

	chartVersion, _ := req.GetString("chart_version")
	cfg.ChartVersion = chartVersion

	helmValues, err := req.GetStringMap("helm_values")
	if err != nil {
		return nil, sdk.NewErrFailed("invalid helm values")
	}
	cfg.HelmValues = helmValues

	cfg.checkStatus, _ = req.GetString("previous", "check_status")

	return cfg, nil

}

func deploy(logger sdk.Logger, cfg *Config) (sdk.Response, error) {
	logger.Info("deploying application")

	cp, err := pullChart(cfg)
	if err != nil {
		return nil, sdk.NewErrTransient("failed to pull chart")
	}

	existingRelease, err := action.NewGet(cfg.actionConfig).Run(cfg.Release)
	if err == nil && existingRelease != nil {
		// Release exists, so we will upgrade it
		existingRelease, err = upgrade(logger, cfg, cp)
		if err != nil {
			return nil, err
		}
	} else {
		// Release does not exist, so we will install it
		existingRelease, err = install(logger, cfg, cp)
		if err != nil {
			return nil, err
		}
	}

	resp := make(sdk.Response)
	resp["release"] = existingRelease

	return resp, nil
}

func destroy(logger sdk.Logger, cfg *Config) (sdk.Response, error) {
	logger.Info("destroying application")
	uninstallClient := action.NewUninstall(cfg.actionConfig)
	uninstallClient.Timeout = time.Minute * 5

	rel, err := uninstallClient.Run(cfg.Release)
	if err != nil {
		return nil, sdk.NewErrTransient(fmt.Sprintf("Failed to uninstall release: %v", err))
	}

	resp := make(sdk.Response)
	resp["uninstall_release"] = rel

	logger.Info("Release uninstalled successfully\n", "release", cfg.Release)
	return resp, nil
}

func install(logger sdk.Logger, cfg *Config, cp string) (*release.Release, error) {
	logger.Info("Release does not exist. Installing...\n", "release", cfg.Release)
	installClient := h3a.NewInstall(cfg.actionConfig)
	installClient.Namespace = cfg.Namespace
	installClient.ReleaseName = cfg.Release
	installClient.RepoURL = cfg.RepoURL
	installClient.Version = cfg.ChartVersion
	installClient.Timeout = time.Minute * 5

	chartPath := cp
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, sdk.NewErrTransient(fmt.Sprintf("Failed to load chart: %v", err))
	}

	vals := map[string]interface{}{} // Add your values here if needed
	rel, err := installClient.Run(chartRequested, vals)
	if err != nil {
		return nil, sdk.NewErrTransient(fmt.Sprintf("Failed to install release: %v", err))
	}

	logger.Info("Release installed successfully\n", "release", cfg.Release)
	return rel, nil
}

func upgrade(logger sdk.Logger, cfg *Config, cp string) (*release.Release, error) {
	logger.Info("Release exists. Upgrading...\n", "release", cfg.Release)
	upgradeClient := action.NewUpgrade(cfg.actionConfig)
	upgradeClient.Namespace = cfg.settings.Namespace()
	upgradeClient.Timeout = time.Minute * 5

	chartPath := cp
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, sdk.NewErrTransient(fmt.Sprintf("Failed to load chart: %v", err))
	}

	vals := cfg.HelmValues
	rel, err := upgradeClient.Run(cfg.Release, chartRequested, vals)
	if err != nil {
		return nil, sdk.NewErrTransient(fmt.Sprintf("Failed to upgrade release: %v", err))
	}

	logger.Info("Release upgraded successfully\n", "release", cfg.Release)
	return rel, nil
}

func pullChart(cfg *Config) (string, error) {
	installClient := h3a.NewInstall(cfg.actionConfig)
	installClient.Namespace = cfg.Namespace
	installClient.ReleaseName = cfg.Release
	installClient.RepoURL = cfg.RepoURL
	installClient.Version = cfg.ChartVersion

	chartName := cfg.RepoName
	if h3r.IsOCI(cfg.RepoURL) {
		registryClient, err := newRegistryClient("", "", "", false, false, cfg.settings)
		if err != nil {
			return "", sdk.NewErrTransient("failed to create registry client")
		}
		installClient.SetRegistryClient(registryClient)
		chartName = cfg.RepoURL
		installClient.RepoURL = ""
	}

	return installClient.ChartPathOptions.LocateChart(chartName, cfg.settings)
}

func createKubeconfigFile(cfg *Config) (string, error) {
	// create tmp kubeconfig file
	kubeconfigFile, err := os.CreateTemp(cfg.tmpDir, "kubeconfig-*.yaml")
	if err != nil {
		return "", sdk.NewErrFailed("failed to create kubeconfig file")
	}

	_, err = kubeconfigFile.WriteString(cfg.Kubeconfig)
	if err != nil {
		return "", sdk.NewErrFailed("failed to write kubeconfig file")
	}
	return kubeconfigFile.Name(), nil
}

func newRegistryClient(certFile, keyFile, caFile string, insecureSkipTLSverify, plainHTTP bool, settings *h3cli.EnvSettings) (*registry.Client, error) {
	if certFile != "" && keyFile != "" || caFile != "" || insecureSkipTLSverify {
		registryClient, err := newRegistryClientWithTLS(certFile, keyFile, caFile, insecureSkipTLSverify, settings)
		if err != nil {
			return nil, err
		}
		return registryClient, nil
	}
	registryClient, err := newDefaultRegistryClient(plainHTTP, settings)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

func newDefaultRegistryClient(plainHTTP bool, settings *h3cli.EnvSettings) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}
	if plainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

func newRegistryClientWithTLS(certFile, keyFile, caFile string, insecureSkipTLSverify bool, settings *h3cli.EnvSettings) (*registry.Client, error) {
	// Create a new registry client
	registryClient, err := registry.NewRegistryClientWithTLS(os.Stderr, certFile, keyFile, caFile, insecureSkipTLSverify,
		settings.RegistryConfig, settings.Debug,
	)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}
