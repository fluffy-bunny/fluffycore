package common

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_services_auth_FinalAuthVerificationServerOptionAccessor_nilverification "github.com/fluffy-bunny/fluffycore/services/auth/FinalAuthVerificationServerOptionAccessor/nilverification"
	fluffycore_services_common_cache "github.com/fluffy-bunny/fluffycore/services/common/cache"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_services_common_time "github.com/fluffy-bunny/fluffycore/services/common/time"
	fluffycore_services_tasks "github.com/fluffy-bunny/fluffycore/services/tasks"
)

func AddCommonServices(builder di.ContainerBuilder) {
	fluffycore_services_common_time.AddTimeNow(builder)
	fluffycore_services_common_time.AddTimeParse(builder)
	fluffycore_services_common_time.AddSingletonITime(builder)
	fluffycore_services_common_time.AddSingletonITimeUtils(builder)
	fluffycore_services_common_claimsprincipal.AddClaimsPrincipal(builder)
	fluffycore_services_common_cache.AddMemoryCache(builder)
	fluffycore_services_tasks.AddTasksServices(builder)
	fluffycore_services_auth_FinalAuthVerificationServerOptionAccessor_nilverification.AddFinalAuthVerificationServerOptionAccessor(builder)
}
