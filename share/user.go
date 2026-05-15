package share

type UserResult struct {
	Changed bool `json:"changed"`
	WouldChange bool `json:"would_change,omitempty"`
	MissingGroups []string `json:"missing_groups,omitempty"`
	Error string `json:"error,omitempty"`
}

type UserConfig struct {
	F_name string `json:"name"`
	F_groups []string `json:"groups"`
	F_append bool `json:"append"`
}
