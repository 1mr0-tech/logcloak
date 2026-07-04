package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/1mr0-tech/logcloak/pkg/masker"
	"github.com/1mr0-tech/logcloak/pkg/metrics"
	"github.com/1mr0-tech/logcloak/pkg/rules"
	"github.com/1mr0-tech/logcloak/pkg/sentinel"
)

const (
	fifoPipe     = "/masker-pipe/app.pipe"
	maxLineBytes = 1 << 20 // 1 MiB
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	metrics.MustRegister()

	podName := os.Getenv("POD_NAME")
	podNS := os.Getenv("POD_NAMESPACE")
	if podName == "" {
		podName = "unknown"
	}

	compiled, err := rules.Deserialize(os.Getenv("LOGCLOAK_RULES"))
	if err != nil {
		logger.Error("failed to parse rules", "error", err)
		fifo, ferr := os.Open(fifoPipe)
		if ferr != nil {
			logger.Error("cannot open FIFO in drop-all mode", "error", ferr)
			os.Exit(1)
		}
		defer fifo.Close() //nolint:errcheck
		dropPipe(fifo, os.Stdout, podName, podNS, "rules_parse_error")
		return
	}

	m := masker.New(compiled)

	go serveMetrics(logger)

	fifo, err := os.Open(fifoPipe)
	if err != nil {
		logger.Error("failed to open FIFO", "path", fifoPipe, "error", err)
		os.Exit(1)
	}
	defer fifo.Close() //nolint:errcheck

	if err := processPipe(fifo, os.Stdout, m, podName, podNS); err != nil {
		logger.Error("scanner error", "error", err)
		os.Exit(1)
	}
}

// processPipe reads lines from r, applies masking, and writes results to w.
// Used by main (with a real FIFO) and by tests (with in-memory readers/writers).
func processPipe(r io.Reader, w io.Writer, m *masker.Masker, podName, podNS string) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxLineBytes), maxLineBytes)
	for scanner.Scan() {
		line := scanner.Text()
		start := time.Now()
		masked, matched := m.MaskLine(line)
		metrics.ProcessingDuration.WithLabelValues(podName, podNS).Observe(time.Since(start).Seconds())
		fmt.Fprintln(w, masked)
		metrics.ProcessedLines.WithLabelValues(podName, podNS).Inc()
		for _, name := range matched {
			metrics.MaskedLines.WithLabelValues(podName, podNS, name).Inc()
		}
	}
	return scanner.Err()
}

// dropPipe reads every line from r and emits a drop sentinel to w.
// Used when rules cannot be parsed so raw PII never reaches stdout.
func dropPipe(r io.Reader, w io.Writer, podName, podNS, reason string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxLineBytes), maxLineBytes)
	for scanner.Scan() {
		fmt.Fprintln(w, sentinel.Line(reason, podName))
		metrics.DroppedLines.WithLabelValues(podName, podNS, reason).Inc()
	}
}

func serveMetrics(logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9090"
	}
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Error("metrics server error", "error", err)
	}
}
