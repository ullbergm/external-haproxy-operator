package haproxyclient

import (
	"github.com/haproxytech/client-native/v6/models"
)

// HAProxyClient defines the main interface for HAProxy operations
type HAProxyClient interface {
	BackendManager
	FrontendManager
	ServerManager
	HTTPCheckManager
	BindManager
	BackendSwitchingRuleManager
	VersionManager
	StartTransaction() (Transaction, error)
	CommitTransaction(id string, force bool) (Transaction, error)
	DeleteTransaction(id string) error
}

// VersionManager handles configuration version operations
type VersionManager interface {
	GetConfigVersion() (int64, error)
}

// BackendManager handles backend operations
type BackendManager interface {
	GetBackend(name string) (*models.Backend, error)
	EnsureBackend(backend *models.Backend) error
	ListBackends() ([]*models.Backend, error)
	DeleteBackend(name string) error
}

// FrontendManager handles frontend operations
type FrontendManager interface {
	GetFrontend(name string) (*models.Frontend, error)
	EnsureFrontend(frontend *models.Frontend) error
}

// ServerManager handles server operations
type ServerManager interface {
	GetServer(backend, name string) (*models.Server, error)
	EnsureServer(backend string, server *models.Server) error
	ListServers(backendName string) ([]*models.Server, error)
	DeleteServer(backendName, serverName string) error
}

// HTTPCheckManager handles HTTP check operations
type HTTPCheckManager interface {
	EnsureBackendHTTPCheck(backend string, httpChecks []*models.HTTPCheck) error
}

// BindManager handles bind operations
type BindManager interface {
	EnsureBind(frontend string, bind *models.Bind) error
}

// BackendSwitchingRuleManager handles backend switching rule operations
type BackendSwitchingRuleManager interface {
	ListBackendSwitchingRules(frontend string) ([]*models.BackendSwitchingRule, error)
	EnsureBackendSwitchingRule(frontend string, rule *models.BackendSwitchingRule) error
	DeleteBackendSwitchingRuleByIndex(frontend string, index int64) error
}
