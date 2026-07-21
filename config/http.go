package config

// HTTPConfig holds the HTTP server connection parameters.
type HTTPConfig struct {
	// Addr is the address the server listens on (e.g. ":8080").
	Addr string
}

// LoadHTTP reads HTTP server settings from environment variables,
// falling back to sensible local-dev defaults when they are absent.
//
//	HTTP_ADDR — listen address (default: :9090)
func LoadHTTP() HTTPConfig {
	return HTTPConfig{
		Addr: env("HTTP_ADDR", ":9090"),
	}
}
