package services

import (
	"database/sql"
	"fmt"
)

// Service defines the interface that all services must implement
type Service interface {
	// Initialize the service with database connection
	Initialize(db *sql.DB) error

	// Name returns the service name for logging
	Name() string

	// Health check (optional, for future use)
	Health() error
}

// ServiceRegistry manages all services
type ServiceRegistry struct {
	services []Service
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make([]Service, 0),
	}
}

// Register adds a service to the registry
func (sr *ServiceRegistry) Register(service Service) {
	sr.services = append(sr.services, service)
}

// InitializeAll initializes all registered services with the database connection
func (sr *ServiceRegistry) InitializeAll(db *sql.DB) error {
	for _, service := range sr.services {
		if err := service.Initialize(db); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", service.Name(), err)
		}
	}
	return nil
}

// GetServices returns all registered services (for handlers to access)
func (sr *ServiceRegistry) GetServices() []Service {
	return sr.services
}
