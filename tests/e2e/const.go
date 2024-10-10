package e2e

const (
	ShootNamespace             = "shoot--logging--test"
	SeedNamespace              = "seed--logging--test"
	BackendContainerImage      = "ghcr.io/credativ/vali:v2.2.18"
	LogGeneratorContainerImage = "nickytd/log-generator:0.1.0"
	DaemonSetName              = "fluent-bit"
	EventLoggerName            = "event-logger"
	SeedBackendName            = "seed"
	ShootBackendName           = "shoot"
)
