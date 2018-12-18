package main

import (
	"time"

	"github.com/kyma-project/kyma/components/application-operator/pkg/client/clientset/versioned/scheme"
	"github.com/kyma-project/kyma/components/application-operator/pkg/controller"
	"github.com/kyma-project/kyma/components/application-operator/pkg/kymahelm"
	appRelease "github.com/kyma-project/kyma/components/application-operator/pkg/kymahelm/application"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	log.Info("Starting Application Operator.")

	options := parseArgs()
	log.Infof("Options: %s", options)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	syncPeriod := time.Second * time.Duration(options.syncPeriod)
	mgrOpts := manager.Options{
		SyncPeriod: &syncPeriod,
	}

	mgr, err := manager.New(cfg, mgrOpts)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Registering Components:")

	log.Printf("Setting up scheme.")
	
	scheme.AddToScheme(mgr.GetScheme())

	log.Printf("Preparing Release Manager.")

	releaseManager, err := newReleaseManager(options)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Setting up controller.")

	err = controller.InitApplicationController(mgr, releaseManager, options.appName)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting the Cmd.")
	log.Info(mgr.Start(signals.SetupSignalHandler()))
}

func newReleaseManager(options *options) (appRelease.ReleaseManager, error) {
	overridesData := appRelease.OverridesData{
		DomainName:             options.domainName,
		ProxyServiceImage:      options.proxyServiceImage,
		EventServiceImage:      options.eventServiceImage,
		EventServiceTestsImage: options.eventServiceTestsImage,
	}

	overrides, err := kymahelm.ParseOverrides(overridesData, appRelease.OverridesTemplate)
	if err != nil {
		return nil, err
	}

	helmClient := kymahelm.NewClient(options.tillerUrl, options.installationTimeout)
	releaseManager := appRelease.NewReleaseManager(helmClient, overrides, options.namespace)

	return releaseManager, nil
}
