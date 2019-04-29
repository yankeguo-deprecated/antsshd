package main

type EndpointRequestProxy struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type EndpointRequestForward struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type EndpointRequest struct {
	Hostname  string                 `json:"hostname"`
	User      string                 `json:"user"`
	PublicKey string                 `json:"public_key"`
	Type      string                 `json:"type"`
	Proxy     EndpointRequestProxy   `json:"proxy"`
	Forward   EndpointRequestForward `json:"forward"`
}
