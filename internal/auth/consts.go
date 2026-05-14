package auth

const (
	ScopeRead       = "read"
	ScopeWritePlans = "write:plans"

	DefaultClientID = "project-brain-client"
)

var DefaultScopes = []string{ScopeRead, ScopeWritePlans}
