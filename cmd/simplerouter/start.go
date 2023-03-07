/*
   Copyright 2021 The simple-router Authors.
   Licensed under the Apache License, PROJECT_VERSION 2.0 (the "License");
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
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/l0calh0st/simple-router/cmd/simplerouter/options"
	"github.com/l0calh0st/simple-router/pkg/controller"
	"github.com/l0calh0st/simple-router/pkg/controller/simple"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
)

func NewStartCommand(stopCh <-chan struct{}) *cobra.Command {
	opts := options.NewOptions()
	cmd := &cobra.Command{
		Short: "Launch simple-router",
		Long:  "Launch simple-router",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return fmt.Errorf("Options validate failed, %v. ", err)
			}
			if err := opts.Complete(); err != nil {
				return fmt.Errorf("Options Complete failed %v. ", err)
			}
			if err := runCommand(opts, stopCh); err != nil {
				return fmt.Errorf("Run %s failed. ", os.Args[0])
			}
			return nil
		},
	}
	fs := cmd.Flags()
	nfs := opts.NamedFlagSets()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(local)
	nfs.FlagSet("logging").AddGoFlagSet(local)

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, cols)
	})
	return cmd
}

func runCommand(o *options.Options, signalCh <-chan struct{}) error {
	klog.Infof("runCommand %v", o)
	var err error
	var stopCh = make(chan struct{})

	register := prometheus.NewRegistry()
	restConfig, err := buildKubeConfig("", "")
	klog.Infof("build KubeConfig successfully")

	if err != nil {
		return err
	}

	kubeClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	klog.Infof("build kubeclient successfully")
	kubeInformers := buildKubeStandardResourceInformerFactory(kubeClientSet)

	simpleController := simple.NewController(o.Config(), kubeClientSet, kubeInformers, register)

	kubeInformers.Start(stopCh)
	if err := runController(stopCh, simpleController); err != nil {
		return fmt.Errorf("run controller failed: %v", err)
	}

	select {
	case <-signalCh:
		klog.Infof("exited")
		close(stopCh)
	case <-stopCh:
	}

	simpleController.Stop()

	return nil
}
func runController(stopCh <-chan struct{}, controller controller.Controller) error {
	if err := controller.Start(1, stopCh); err != nil {
		return err
	}
	return nil

}

func serve(srv *http.Server, listener net.Listener) func() error {
	return func() error {
		//level.Info(logger).Log("msg", "Starting insecure server on "+listener.Addr().String())
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func serveTLS(srv *http.Server, listener net.Listener) func() error {
	return func() error {
		//level.Info(logger).Log("msg", "Starting secure server on "+listener.Addr().String())
		if err := srv.ServeTLS(listener, "", ""); err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// buildKubeConfig build rest.Config from the following ways
// 1: path of kube_config 2: KUBECONFIG environment 3. ~/.kube/config, as kubeconfig may not in /.kube/
func buildKubeConfig(masterUrl, kubeConfig string) (*rest.Config, error) {
	cfgLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfgLoadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	cfgLoadingRules.ExplicitPath = kubeConfig
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(cfgLoadingRules, &clientcmd.ConfigOverrides{})
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	if err = rest.SetKubernetesDefaults(config); err != nil {
		return nil, err
	}
	return config, nil
}

// buildKubeStandardResourceInformerFactory build a kube informer factory according some options
func buildKubeStandardResourceInformerFactory(kubeClient kubernetes.Interface) informers.SharedInformerFactory {
	var factoryOpts []informers.SharedInformerOption
	factoryOpts = append(factoryOpts, informers.WithNamespace(apicorev1.NamespaceAll))
	return informers.NewSharedInformerFactoryWithOptions(kubeClient, 5*time.Second, factoryOpts...)
}
