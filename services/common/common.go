package common

import (
	di "github.com/dozm/di"
	fluffycore_services_common_cache "github.com/fluffy-bunny/fluffycore/services/common/cache"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_services_common_time "github.com/fluffy-bunny/fluffycore/services/common/time"
)

func AddCommonServices(builder di.ContainerBuilder) {
	fluffycore_services_common_time.AddTimeNow(builder)
	fluffycore_services_common_time.AddTimeParse(builder)
	fluffycore_services_common_time.AddSingletonITime(builder)
	fluffycore_services_common_time.AddSingletonITimeUtils(builder)
	fluffycore_services_common_claimsprincipal.AddClaimsPrincipal(builder)
	fluffycore_services_common_cache.AddMemoryCache(builder)
}
