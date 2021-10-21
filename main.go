/*
Copyright 2021 xiayuhai.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"path/filepath"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkv1 "wooshnet/apis/network/v1"
	networkcontrollers "wooshnet/controllers/network"

	"wooshnet/pkg/deviceclient"
	"wooshnet/pkg/netclient"
	"wooshnet/pkg/server"
	"wooshnet/pkg/wooshtools"

	netdef "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(networkv1.AddToScheme(scheme))
	utilruntime.Must(netdef.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	var namespace string
	var mode string
	var nodeName string
	var unixSock string
	flag.StringVar(&namespace, "namespace", os.Getenv("SYSTEM_NAMESPACE"), "The namespace woosh run.")
	flag.StringVar(&mode, "mode", "controller", "mode: controller/daemon")
	flag.StringVar(&nodeName, "nodeName", os.Getenv("NODE_NAME"), "k8s nodeName")
	flag.StringVar(&unixSock, "unixSock", "/var/run/wooshnet/woosh.sock", "unix socket for daemon to listen, support for cni")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "133e1de2.wooshnet.cn",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	key := types.NamespacedName{
		Name:      "woosh-config",
		Namespace: namespace,
	}
	obj := &corev1.ConfigMap{}
	err = mgr.GetAPIReader().Get(context.Background(), key, obj)
	if err != nil {
		setupLog.Error(err, "unable to read config")
		os.Exit(1)
	}
	neutron, err := netclient.NewNeutronClient(obj)
	if err != nil {
		setupLog.Error(err, "unable to create neutron client")
		os.Exit(1)
	}
	var deviceClient deviceclient.DeviceClient
	if mode == "controller" {
		if err = (&networkcontrollers.PodReconciler{
			Client:          mgr.GetClient(),
			Scheme:          mgr.GetScheme(),
			SystemNamespace: namespace,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Pod")
			os.Exit(1)
		}
		if err = (&networkcontrollers.ConfigMapReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ConfigMap")
			os.Exit(1)
		}
		if err = (&networkcontrollers.VifPoolReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			NetClient: neutron,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "VifPool")
			os.Exit(1)
		}
	} else {
		deviceClient, err = deviceclient.NewOvsClient(obj)
		if err != nil {
			setupLog.Error(err, "unable to create neutron client")
			os.Exit(1)
		}
	}
	if err = (&networkcontrollers.WooshPortReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		NetClient:    neutron,
		DeviceClient: deviceClient,
		Mode:         mode,
		NodeName:     nodeName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WooshPort")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	setupLog.Info("starting manager")
	if mode == "daemon" {
		go Listen(ctx, unixSock, mgr.GetClient())
	}
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func Listen(ctx context.Context, unixsock string, k8scli client.Client) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	logger.Info("Listening and serving HTTP on unix://" + unixsock)

	os.MkdirAll(filepath.Dir(unixsock), 0755)
	if wooshtools.FileExist(unixsock) {
		os.Remove(unixsock)
	}
	l, err := net.Listen("unix", unixsock)
	if err != nil {
		logger.Error(err, unixsock)
		os.Exit(1)
	}
	defer l.Close()
	defer os.Remove(unixsock)

	handler := gin.New()
	handler.Use(gin.Logger(), gin.Recovery())

	k8sconf := &server.ServerConfig{
		K8sCli: k8scli,
	}
	k8sconf.Register(handler)
	os.Remove(unixsock)
	err = handler.RunUnix(unixsock)
	if err != nil {
		logger.Error(err, unixsock)
		os.Exit(1)
	}
	return nil
}
