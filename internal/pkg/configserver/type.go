package configserver

// AgentGroup represents an agent group
type AgentGroup struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}