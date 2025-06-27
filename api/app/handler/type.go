package handler

type httpResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type clientGroup []string
type clientGroupResponse struct {
	Groups map[string][]string `json:"groups"`
}
