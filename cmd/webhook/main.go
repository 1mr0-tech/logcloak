package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	logger.Info("starting", "component", "webhook", "version", version)
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
		logger.Error("in-cluster config", "error", err)
		os.Exit(1)
	}

	kube, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		logger.Error("kubernetes client", "error", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := rules.AddToScheme(scheme); err != nil {
		logger.Error("add scheme", "error", err)
		os.Exit(1)
	}

	ctrlClient, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("controller-runtime client", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	tlsMgr, err := webhook.NewTLSManager(ctx, kube, namespace, serviceName, logger)
	if err != nil {
		logger.Error("TLS setup", "error", err)
		os.Exit(1)
	}

	if err := webhook.PatchWebhookCABundle(ctx, kube, webhookName, tlsMgr.CACert()); err != nil {
		logger.Warn("patch webhook caBundle", "error", err)
	}

	go tlsMgr.WatchAndRotate(ctx, webhookName)

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
		Addr:              ":8443",
		Handler:           mux,
		TLSConfig:         tlsMgr.Config(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", ":8443", "tls", true)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	go func() {
		logger.Info("metrics listening", "addr", ":9090")
		if err := http.ListenAndServe(":9090", metricsMux); err != nil {
			logger.Error("metrics server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
