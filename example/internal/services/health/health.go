package health

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_services_health "github.com/fluffy-bunny/fluffycore/services/health"
)

// AddHealthService registers fluffycore's health service. Replace this with your own
// IHealthServer to report application-specific health.
func AddHealthService(cb di.ContainerBuilder) {
	fluffycore_services_health.AddHealthService(cb)
}
