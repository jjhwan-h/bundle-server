package handler

// type hookClient struct {
// 	Addr string `json:"addr"`
// }

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
