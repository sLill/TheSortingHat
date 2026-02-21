package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// updateResponse is the JSON body returned to Tauri v2 when an update is available.
// Tauri expects exactly these fields for a per-platform update endpoint.
type updateResponse struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	PubDate   string `json:"pub_date"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// Handler handles Tauri update check requests.
type Handler struct {
	store *ConfigStore
}

// NewHandler returns a Handler backed by store.
func NewHandler(store *ConfigStore) *Handler {
	return &Handler{store: store}
}

// ServeHTTP handles GET /update/{target}-{arch}/{current_version}
//
// Response:
//   - 204 No Content  — client is up to date or not eligible for any pending release
//   - 200 OK          — JSON body with the highest eligible release for this platform
//   - 400 Bad Request — malformed path or unparseable current version
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip the /update/ prefix, leaving "{target}-{arch}/{current_version}"
	trimmed := strings.TrimPrefix(r.URL.Path, "/update/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "path must be /update/{target}-{arch}/{current_version}", http.StatusBadRequest)
		return
	}

	platform := parts[0]        // e.g. "linux-x86_64", "darwin-aarch64", "windows-x86_64"
	currentVersionStr := parts[1]

	customer := r.Header.Get("X-CUSTOMER")
	region := r.Header.Get("X-REGION")
	machineID := r.Header.Get("X-MACHINE-ID")

	currentVersion, err := semver.NewVersion(currentVersionStr)
	if err != nil {
		http.Error(w, "invalid current_version: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := h.store.Get()

	// Walk all releases to find the highest version the client is eligible for.
	var bestRelease *Release
	var bestVersion *semver.Version

	for i := range cfg.Releases {
		rel := &cfg.Releases[i]

		v, err := semver.NewVersion(rel.Version)
		if err != nil {
			log.Printf("skipping release with invalid version %q: %v", rel.Version, err)
			continue
		}

		// Must be strictly newer than what the client already has.
		if !v.GreaterThan(currentVersion) {
			continue
		}

		// Must include an asset for the client's platform.
		if _, ok := rel.Platforms[platform]; !ok {
			continue
		}

		// Must satisfy rollout rules for this client.
		if !eligible(rel.Rollout, customer, region, machineID, rel.Version) {
			continue
		}

		// Keep the highest eligible version.
		if bestVersion == nil || v.GreaterThan(bestVersion) {
			bestRelease = rel
			bestVersion = v
		}
	}

	if bestRelease == nil {
		// No eligible update — tell Tauri there is nothing to install.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	asset := bestRelease.Platforms[platform]
	resp := updateResponse{
		Version:   bestRelease.Version,
		Notes:     bestRelease.Notes,
		PubDate:   bestRelease.PubDate,
		URL:       asset.URL,
		Signature: asset.Signature,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}
