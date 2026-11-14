package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/dubstepenjoyer/cashmere/internal/cache"
)

type CacheService interface {
	Get(ctx context.Context, key cache.Key) (cache.Result, error)
	Delete(ctx context.Context, key cache.Key) error
}

type Handler struct {
	service CacheService
	logger  *slog.Logger
}

func New(service CacheService, logger *slog.Logger) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("httpapi: cache service is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	h := &Handler{
		service: service,
		logger:  logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/cache/{provider}", h.get)
	mux.HandleFunc("DELETE /v1/cache/{provider}", h.delete)

	return mux, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "empty key", http.StatusBadRequest)
		return
	}

	cackeKey := cache.Key{
		Provider: provider,
		ID:       key,
	}

}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {

}
