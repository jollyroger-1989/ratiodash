package service

import (
	"github.com/jose/ratiodash/internal/domain"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewTrackerService),
	fx.Provide(NewStatsService),
	fx.Provide(NewAlertConfigService),
	fx.Provide(NewAlertServiceWithAuthRepo),
	fx.Provide(NewRefreshService),
	fx.Provide(NewAuthService),
	fx.Provide(
		fx.Annotate(
			NewAPIClientService,
			fx.As(new(domain.APIClientService)),
			fx.As(new(domain.APIKeyAuthenticator)),
		),
	),
	fx.Provide(NewNotifierConfigService),
	fx.Provide(NewReportServiceWithAuthRepo),
)
