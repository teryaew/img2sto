package main

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/schema"
)

// OptionsMW is a wrapper to provide appContext to options middleware.
type OptionsMW struct {
	ctx *AppContext
}

// Middleware for OptionsMW prepares options from query params.
func (omw *OptionsMW) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := schema.NewDecoder()
		err := decoder.Decode(omw.ctx.Options, r.URL.Query())
		if err != nil {
			renderError(w, err, "RESIZE_PARAMS_ARE_INVALID", http.StatusBadRequest)
		}

		next.ServeHTTP(w, r)
	})
}

// CorsMW provides Cross-Origin Resource Sharing middleware.
func CorsMW(next http.Handler) http.Handler {
	return handlers.CORS()(next)
}

// LoggingMW provides logging middleware.
func LoggingMW(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}
