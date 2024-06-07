//go:generate swag init
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hyperdxio/opentelemetry-go/otelzap"
	"github.com/hyperdxio/opentelemetry-logs-go/exporters/otlp/otlplogs"
	sdk "github.com/hyperdxio/opentelemetry-logs-go/sdk/logs"
	"github.com/hyperdxio/otel-config-go/otelconfig"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	_ "github.com/${{ values.orgName }}/${{ values.repoName}}/docs"
)

var desiredNbObjects = 5
var objectsSizeInMB = 3
var intervalInSecs = 15
var customMessage = "Hello"

var globalSlice []byte
var nbObjects int = 0
var done = make(chan bool)

const defaultPort = "8080"

var logger *zap.Logger

// @title ${{ values.repoName }} API
// @version 1.0
// @description ${{ values.description }}
// @host http://${{ name }}-sreez.apps.oc-med.wk.nt.local
func main() {
	// Initialize otel config and use it across the entire app
	println("Service starting up")

	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	ctx := context.Background()

	// configure opentelemetry logger provider
	logExporter, _ := otlplogs.NewExporter(ctx)
	loggerProvider := sdk.NewLoggerProvider(
		sdk.WithBatcher(logExporter),
	)
	// gracefully shutdown logger to flush accumulated signals before program finish
	defer loggerProvider.Shutdown(ctx)

	// create new logger with opentelemetry zap core and set it globally
	logger = zap.New(otelzap.NewOtelCore(loggerProvider))
	zap.ReplaceGlobals(logger)

	interval := time.Second * time.Duration(intervalInSecs)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Start a goroutine to run the function every interval
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				recurrentFunction(t)
			}
		}
	}()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := Handler()
	logger.Info("** Service Started on Port " + port + " **")
	println("** Service Started on Port " + port + " **")
	if err := http.ListenAndServe(":"+port, otelhttp.NewHandler(r, "example-service")); err != nil {
		logger.Fatal(err.Error())
	}
}

type ExampleResponse struct {
	InstancesCount int    `json:"instances_count"`
	IntervalSec    int    `json:"interval_sec"`
	Message        string `json:"message"`
	RequestId      string `json:"request_id"`
}

// ExampleHandler godoc
// @Summary      Return sample count payload
// @Tags         Example
// @Produce      json
// @Success      200 {object}  ExampleResponse
// @Router       / [get]
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	id := middleware.GetReqID(r.Context())
	payload, err := json.Marshal(&ExampleResponse{
		InstancesCount: nbObjects,
		IntervalSec:    intervalInSecs,
		Message:        customMessage,
		RequestId:      id,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("{\"error\": \"%s\"}", err))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
	return
}

func Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.SupressNotFound(r))
	r.Use(middleware.Logger)
	r.Use(middleware.Heartbeat("/health"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/", wrapHandler(logger, ExampleHandler))
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
			SpecURL:  "./docs/swagger.json",
			DarkMode: true,
		})

		if err != nil {
			fmt.Printf("%v", err)
		}

		fmt.Fprintln(w, htmlContent)
	})
	return r
}

// attach trace id to the log
func withTraceMetadata(ctx context.Context, logger *zap.Logger) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		// ctx does not contain a valid span.
		// There is no trace metadata to add.
		return logger
	}
	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}

// Use this to wrap all handlers to add trace metadata to the logger
func wrapHandler(logger *zap.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := withTraceMetadata(r.Context(), logger)
		logger.Info("request received", zap.String("url", r.URL.Path), zap.String("method", r.Method))
		handler(w, r)
		logger.Info("request completed", zap.String("path", r.URL.Path), zap.String("method", r.Method))
	}
}

func recurrentFunction(t time.Time) {
	formattedTime := t.Format("2006-01-02 15:04:05")
	fmt.Printf("%v: Allocated objects: %d\n", formattedTime, nbObjects)
	if nbObjects < desiredNbObjects {
		fmt.Printf("%v: Allocating new object\n", formattedTime)
		data := make([]byte, 1024*1024*objectsSizeInMB)
		globalSlice = append(globalSlice, data...)
		nbObjects++
	} else {
		fmt.Printf("%v: Objects limit reached (%d), no new allocation, stopping ticker\n", formattedTime, desiredNbObjects)
		done <- true
	}
}

// configure common attributes for all logs
func newResource() *resource.Resource {
	hostName, _ := os.Hostname()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceVersion("1.0.0"),
		semconv.HostName(hostName),
	)
}
