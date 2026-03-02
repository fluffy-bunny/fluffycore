package plugin

import (
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
)

type (
	RegisterService func(builder di.ContainerBuilder)
	ServicePlugin   struct {}
)

var ServiceRegistrations = []RegisterService{}
var lock = sync.Mutex{}

func RegisterServiceRegistration(registerService RegisterService) {
	lock.Lock()
	defer lock.Unlock()
	ServiceRegistrations = append(ServiceRegistrations, registerService)
}
