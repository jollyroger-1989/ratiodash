package handler

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(NewTrackerHandler),
	fx.Provide(NewStatsHandler),
	fx.Provide(NewScraperHandler),
	fx.Provide(NewAuthHandler),
	fx.Provide(NewNotifierConfigHandler),
	fx.Provide(NewReportHandler),
	fx.Provide(NewAlertConfigHandler),
	fx.Invoke(RegisterTrackerRoutes),
	fx.Invoke(RegisterStatsRoutes),
	fx.Invoke(RegisterScraperRoutes),
	fx.Invoke(RegisterAuthRoutes),
	fx.Invoke(RegisterNotifierConfigRoutes),
	fx.Invoke(RegisterReportRoutes),
	fx.Invoke(RegisterAlertConfigRoutes),
)
