package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

var httpRequestsTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	},
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	httpRequestsTotal.Inc()
	tracer := otel.Tracer("go-service")
	ctx, span := tracer.Start(r.Context(), "hello-handler")
	defer span.End()
	fmt.Fprintln(w, "Hello Go")
	_ = ctx
}

// Health route
func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func main() {
	prometheus.MustRegister(httpRequestsTotal)

	ctx := context.Background()

	// OTLP/gRPC exporter — matching python-service
	exp, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	tp := trace.NewTracerProvider(trace.WithBatcher(exp))
	otel.SetTracerProvider(tp)

	// Flush pending spans on shutdown
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			fmt.Println("error shutting down tracer provider:", err)
		}
	}()

	// App route
	http.HandleFunc("/go", helloHandler)

	// Health route
	http.HandleFunc("/health", healthHandler)

	// Metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: "0.0.0.0:8000"}

	go func() {
		fmt.Println("Go service running on port 8000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("server error:", err)
		}
	}()

	// Wait for interrupt/terminate to shut down gracefully (flushes traces via defer above)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
