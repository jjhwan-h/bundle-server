package handler

type patch struct {
	Data []patchData `json:"data"`
}

type patchData struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type httpResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type clientGroup []string
type clientGroupResponse struct {
	Groups map[string][]string `json:"groups"`
}
