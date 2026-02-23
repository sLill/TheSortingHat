package main

import (
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// PlatformAsset holds the download URL and detached signature for a specific platform build.
type PlatformAsset struct {
	URL       string `yaml:"url"`
	Signature string `yaml:"signature"`
}

// Rollout controls which clients are eligible for a release.
// All conditions use OR logic: a client is eligible if it satisfies ANY listed condition.
// If Rollout is nil (omitted from config), the release is served to everyone.
type Rollout struct {
	// Customers is an exact-match whitelist against the X-CUSTOMER header.
	Customers []string `yaml:"customers"`
	// Regions is an exact-match whitelist against the X-REGION header.
	Regions []string `yaml:"regions"`
	// Percentage, if non-nil, enables deterministic percentage-based rollout.
	// The bucket is derived from hash(X-MACHINE-ID + ":" + version) % 100.
	// A value of 0 means 0 % of machines receive this update via this rule.
	// A value of 100 means all machines receive it (equivalent to no percentage rule).
	Percentage *int `yaml:"percentage"`
}

// Release represents a single version that may be served to clients.
type Release struct {
	Version   string                    `yaml:"version"`
	Notes     string                    `yaml:"notes"`
	PubDate   string                    `yaml:"pub_date"`
	Platforms map[string]PlatformAsset  `yaml:"platforms"`
	Rollout   *Rollout                  `yaml:"rollout"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `yaml:"port"`
}

// Config is the top-level structure of the YAML config file.
type Config struct {
	Server   ServerConfig `yaml:"server"`
	Releases []Release    `yaml:"releases"`
}

// ConfigStore holds the current config and provides thread-safe access.
type ConfigStore struct {
	mu     sync.RWMutex
	config Config
	path   string
}

// NewConfigStore loads the config at path and returns a ConfigStore.
func NewConfigStore(path string) (*ConfigStore, error) {
	cs := &ConfigStore{path: path}
	if err := cs.load(); err != nil {
		return nil, err
	}
	return cs, nil
}

func (cs *ConfigStore) load() error {
	data, err := os.ReadFile(cs.path)
	if err != nil {
		return err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	cs.mu.Lock()
	cs.config = cfg
	cs.mu.Unlock()
	log.Printf("config loaded: %d release(s)", len(cfg.Releases))
	return nil
}

// Get returns a snapshot of the current config.
func (cs *ConfigStore) Get() Config {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.config
}

// Watch starts a background goroutine that reloads the config whenever the file changes.
func (cs *ConfigStore) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(cs.path); err != nil {
		watcher.Close()
		return err
	}
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Printf("config file changed, reloading")
					if err := cs.load(); err != nil {
						log.Printf("error reloading config: %v", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("fsnotify error: %v", err)
			}
		}
	}()
	return nil
}
