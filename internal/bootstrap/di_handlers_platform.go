package bootstrap

import (
	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
)

func wirePlatformHandlers(h *bootstrapHandlers, d handlerDeps) {
	auditWrite := &usecase.AuditWriteUsecase{
		Repo:  d.repos.auditRepo,
		IDGen: d.idGen,
		Now:   d.now,
	}
	h.platformTenantHandler = &handler.PlatformTenantHandler{
		CreateUC: &usecase.CreatePlatformTenantUsecase{
			Repo:  d.repos.tenantRepo,
			Perm:  d.perm,
			IDGen: d.idGen,
			Now:   d.now,
		},
		ListUC: &usecase.ListPlatformTenantsUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		UpdateUC: &usecase.UpdatePlatformTenantUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
			Now:  d.now,
		},
		ToggleUC: &usecase.TogglePlatformTenantStatusUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		ResetAuthUC: &usecase.ResetPlatformTenantAdminAuthUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		CheckDisplayNameUC: &usecase.CheckPlatformTenantDisplayNameUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		CheckAdminAccountUC: &usecase.CheckPlatformTenantAdminAccountUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		CheckAdminPhoneUC: &usecase.CheckPlatformTenantAdminPhoneUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		ChangeAdminUC: &usecase.ChangePlatformTenantAdminUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		ResetPasswordUC: &usecase.ResetPlatformTenantAdminPasswordUsecase{
			Repo: d.repos.tenantRepo,
			Perm: d.perm,
		},
		AuditUC: auditWrite,
	}
}
