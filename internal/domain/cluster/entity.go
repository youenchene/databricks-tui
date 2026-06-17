// Package cluster defines the cluster browsing domain.
package cluster

import "time"

// State represents a cluster's lifecycle state.
type State string

const (
	StatePending       State = "PENDING"
	StateRunning       State = "RUNNING"
	StateRestarting    State = "RESTARTING"
	StateResizing      State = "RESIZING"
	StateTerminating   State = "TERMINATING"
	StateTerminated    State = "TERMINATED"
	StateError         State = "ERROR"
	StateUnknown       State = "UNKNOWN"
)

// Cluster represents a Databricks cluster in the domain model.
// This is a pure domain type — it has no SDK or framework dependencies.
type Cluster struct {
	ID           string
	Name         string
	State        State
	SparkVersion string
	NodeTypeID   string
	NumWorkers   int32
	Creator      string
	CreatedAt    time.Time
}

// IsAlive returns true when the cluster is running or starting.
func (c Cluster) IsAlive() bool {
	return c.State == StateRunning || c.State == StatePending
}

// Summary returns a one-line human-readable summary.
func (c Cluster) Summary() string {
	return c.Name + " (" + string(c.State) + ")"
}
