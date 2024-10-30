package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.uber.org/zap"

	"github.com/3FT-io/3DS/pkg/core"
	"github.com/3FT-io/3DS/pkg/p2p"
)

type API struct {
	node    *core.Node
	network *p2p.Network
	storage *core.Storage
	logger  *zap.Logger
	server  *http.Server
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewAPI(node *core.Node, network *p2p.Network, storage *core.Storage, port int) (*API, error) {
	logger, _ := zap.NewProduction()

	api := &API{
		node:    node,
		network: network,
		storage: storage,
		logger:  logger,
	}

	router := mux.NewRouter()
	api.setupRoutes(router)

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	api.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return api, nil
}

func (api *API) setupRoutes(router *mux.Router) {
	// Health check
	router.HandleFunc("/health", api.HealthCheck).Methods("GET")

	// Model management
	router.HandleFunc("/models", api.UploadModel).Methods("POST")
	router.HandleFunc("/models", api.ListModels).Methods("GET")
	router.HandleFunc("/models/{id}", api.GetModel).Methods("GET")
	router.HandleFunc("/models/{id}", api.DeleteModel).Methods("DELETE")
	router.HandleFunc("/models/{id}/metadata", api.GetModelMetadata).Methods("GET")

	// Network status
	router.HandleFunc("/network/status", api.GetNetworkStatus).Methods("GET")
	router.HandleFunc("/network/peers", api.GetPeers).Methods("GET")

	// Storage status
	router.HandleFunc("/storage/status", api.GetStorageStatus).Methods("GET")
}

func (api *API) Start() error {
	api.logger.Info("Starting API server", zap.String("addr", api.server.Addr))
	return api.server.ListenAndServe()
}

func (api *API) Stop(ctx context.Context) error {
	return api.server.Shutdown(ctx)
}

// Health check handler
func (api *API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, APIResponse{
		Success: true,
		Data: map[string]string{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// Model upload handler
func (api *API) UploadModel(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		api.sendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("model")
	if err != nil {
		api.sendError(w, "No model file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get model format from filename or form data
	format := r.FormValue("format")
	if format == "" {
		format = getFormatFromFilename(header.Filename)
	}

	// Store the model
	metadata, err := api.storage.StoreModel(r.Context(), header.Filename, format, file)
	if err != nil {
		api.sendError(w, "Failed to store model", http.StatusInternalServerError)
		return
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data:    metadata,
	})
}

// List models handler
func (api *API) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := api.storage.ListModels(r.Context())
	if err != nil {
		api.sendError(w, "Failed to list models", http.StatusInternalServerError)
		return
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data:    models,
	})
}

// Get model handler
func (api *API) GetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	model, err := api.storage.GetModel(r.Context(), modelID)
	if err != nil {
		api.sendError(w, "Model not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers for file download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", model.Name))
	w.Header().Set("Content-Type", getContentType(model.Format))

	if err := api.storage.StreamModel(r.Context(), modelID, w); err != nil {
		api.logger.Error("Failed to stream model", zap.Error(err))
		return
	}
}

// Get model metadata handler
func (api *API) GetModelMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	metadata, err := api.storage.GetModelMetadata(r.Context(), modelID)
	if err != nil {
		api.sendError(w, "Model not found", http.StatusNotFound)
		return
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data:    metadata,
	})
}

// Delete model handler
func (api *API) DeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if err := api.storage.DeleteModel(r.Context(), modelID); err != nil {
		api.sendError(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data: map[string]string{
			"message": "Model deleted successfully",
		},
	})
}

// Network status handler
func (api *API) GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	peers := api.network.GetPeers()

	api.sendResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"peer_count": len(peers),
			"node_id":    api.network.GetHost().ID().String(),
			"addresses":  api.network.GetHost().Addrs(),
		},
	})
}

// Get peers handler
func (api *API) GetPeers(w http.ResponseWriter, r *http.Request) {
	peers := api.network.GetPeers()
	peerInfo := make([]map[string]interface{}, 0, len(peers))

	for _, peer := range peers {
		peerInfo = append(peerInfo, map[string]interface{}{
			"id":        peer.String(),
			"addresses": api.network.GetHost().Peerstore().Addrs(peer),
		})
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data:    peerInfo,
	})
}

// Storage status handler
func (api *API) GetStorageStatus(w http.ResponseWriter, r *http.Request) {
	status, err := api.storage.GetStatus(r.Context())
	if err != nil {
		api.sendError(w, "Failed to get storage status", http.StatusInternalServerError)
		return
	}

	api.sendResponse(w, APIResponse{
		Success: true,
		Data:    status,
	})
}

// Helper functions
func (api *API) sendResponse(w http.ResponseWriter, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *API) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

func getContentType(format string) string {
	switch format {
	case "gltf":
		return "model/gltf+json"
	case "glb":
		return "model/gltf-binary"
	case "obj":
		return "text/plain"
	case "fbx":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func getFormatFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	// Remove the leading dot and convert to lowercase
	return strings.ToLower(ext[1:])
}
