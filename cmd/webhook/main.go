package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/1mr0-tech/logcloak/pkg/metrics"
	"github.com/1mr0-tech/logcloak/pkg/rules"
	"github.com/1mr0-tech/logcloak/pkg/webhook"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "logcloak-webhook %s starting\n", version)
	metrics.MustRegister()

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "logcloak"
	}
	sidecarImage := os.Getenv("SIDECAR_IMAGE")
	if sidecarImage == "" {
		sidecarImage = "ghcr.io/1mr0-tech/logcloak-sidecar:latest"
	}
	webhookName := os.Getenv("WEBHOOK_NAME")
	if webhookName == "" {
		webhookName = "logcloak-webhook"
	}
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "logcloak-webhook"
	}

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "in-cluster config: %v\n", err)
		os.Exit(1)
	}

	kube, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubernetes client: %v\n", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := rules.AddToScheme(scheme); err != nil {
		fmt.Fprintf(os.Stderr, "add scheme: %v\n", err)
		os.Exit(1)
	}

	ctrlClient, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "controller-runtime client: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	tlsCfg, caCert, err := webhook.EnsureTLS(ctx, kube, namespace, serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TLS setup: %v\n", err)
		os.Exit(1)
	}

	if err := webhook.PatchWebhookCABundle(ctx, kube, webhookName, caCert); err != nil {
		fmt.Fprintf(os.Stderr, "patch webhook caBundle: %v (continuing)\n", err)
	}

	h := &webhook.Handler{Client: ctrlClient, SidecarImage: sidecarImage}

	mux := http.NewServeMux()
	mux.Handle("/mutate", h)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsCfg,
	}

	go func() {
		fmt.Fprintf(os.Stderr, "logcloak-webhook listening on :8443\n")
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	fmt.Fprintf(os.Stderr, "shutting down\n")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
