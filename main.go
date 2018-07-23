// img2sto
//
// Image To Storage proxy for uploading/downloading images
// to/from Minio storage with resizing on-the-fly.
//
// 		Schemes: http
// 		Host: localhost:8080
// 		BasePath: /
// 		Version: 1.0.0
//
// swagger:meta
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/caarlos0/env"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
)

const ApiVersion = "v1"
const ServerTimeout = 5 * time.Second

// MIME types
const (
	MIMEApplicationJSON = "application/json"
	MIMEImageJPEG       = "image/jpeg"
	MIMEImageGIF        = "image/gif"
	MIMEImagePNG        = "image/png"
)

type AppContext struct {
	Storage    *minio.Client
	Config     *Config
	Options    *Options
	Dimensions []int
}

type Config struct {
	ImageServiceURL  string `env:"IMAGE_SERVICE_URL"`
	StorageURL       string `env:"STORAGE_URL"`
	StorageAccessKey string `env:"STORAGE_ACCESS_KEY"`
	StorageSecretKey string `env:"STORAGE_SECRET_KEY"`
	StorageSSL       bool   `env:"STORAGE_SSL"`
	Port             string `env:"PORT" envDefault:":8080"`
	ResizeOnUpload   bool   `env:"RESIZE_ON_UPLOAD"`
	ResizeOnDownload bool   `env:"RESIZE_ON_DOWNLOAD"`
}

type Options struct {
	Enlarge bool
	Resize  string
}

func main() {
	log.Println("API:", ApiVersion)
	initServer(CreateContext())
}

// CreateContext returns new app context
func CreateContext() *AppContext {
	// Prepare config
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Config:", cfg)

	// Init app context
	ctx := AppContext{
		Storage: initStorage(&cfg),
		Config:  &cfg,
		Options: &Options{},
	}

	return &ctx
}

func initStorage(cfg *Config) *minio.Client {
	client, err := minio.New(cfg.StorageURL, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageSSL)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}

func initServer(appCtx *AppContext) {
	server := http.Server{
		Addr:         appCtx.Config.Port,
		ReadTimeout:  ServerTimeout,
		WriteTimeout: ServerTimeout,
	}

	// Apply router to server
	server.Handler = InitRouter(appCtx)
	// Serve gracefully
	graceShutdown(&server)
}

// InitRouter initializes & returns new router.
func InitRouter(appCtx *AppContext) http.Handler {
	router := mux.NewRouter()
	subrouter := router.PathPrefix("/" + ApiVersion).Subrouter()

	// Register routes
	router.HandleFunc("/health", HealthController).Methods("GET")
	subrouter.Path("/{bucket}/upload").Handler(&UploadHandler{appCtx}).Methods("POST")
	subrouter.Path("/{bucket}/{object}").Handler(&DownloadHandler{appCtx}).Methods("GET")

	// Apply middlewares
	omw := &OptionsMW{appCtx}
	subrouter.Use(omw.Middleware)
	subrouter.Use(CorsMW)
	router.Use(LoggingMW)

	return router
}

// graceShutdown implements server graceful shutdown
func graceShutdown(server *http.Server) {
	// Subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	go func() {
		log.Printf("Server is listening %s\n", server.Addr)
		// Handle connections
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for SIGINT
	<-stopChan
	log.Println("Shutting down the server...")

	// Shut down gracefully, but wait no longer than timeout before halting
	ctx, cancel := context.WithTimeout(context.Background(), ServerTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server gracefully stopped")
	os.Exit(0)
}

// renderError implements common app error handling
func renderError(w http.ResponseWriter, err error, message string, statusCode int) {
	log.Println(message, err)
	http.Error(w, err.Error(), statusCode)
}
