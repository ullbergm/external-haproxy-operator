package haproxyclient

import "fmt"

// ErrNotManaged is returned when trying to modify a resource not managed by this controller
type ErrNotManaged struct {
	ResourceType string
	ResourceName string
}

func (e ErrNotManaged) Error() string {
	return fmt.Sprintf("%s %q exists but is not managed by this controller", e.ResourceType, e.ResourceName)
}

// ErrResourceNotFound is returned when a resource is not found
type ErrResourceNotFound struct {
	ResourceType string
	ResourceName string
}

func (e ErrResourceNotFound) Error() string {
	return fmt.Sprintf("%s %q not found", e.ResourceType, e.ResourceName)
}

// ErrAPIResponse represents an API error response
type ErrAPIResponse struct {
	StatusCode int
	Body       string
	Operation  string
}

func (e ErrAPIResponse) Error() string {
	return fmt.Sprintf("failed to %s, status: %d, body: %s", e.Operation, e.StatusCode, e.Body)
}
