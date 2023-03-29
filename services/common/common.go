package common

import (
	"github.com/dozm/di"
	fluffycore_services_common_time "github.com/fluffy-bunny/fluffycore/services/common/time"

)
func AddCommonServices(builder di.ContainerBuilder) {
	fluffycore_services_common_time.AddTimeNow(builder)
}
