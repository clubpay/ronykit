---
service_lifecycle: "Service struct with App(), Module(), init() for di.RegisterService, Desc() delegating to api, Start/Shutdown forwarding to fx.App."
module_wiring: "fx.Module wiring: v0repo.Init, diDatasource (di.ProvideDBParams, di.ProvideRedisParams), fx.Provide(settings.New, app.New, api.New), datasource.Init*."
api_contracts: "RContext type alias, ServiceParams (fx.In), Service struct, Desc() grouping handler funcs by domain. Separate api_*.go files for each domain concept with Input/Output DTOs and thin handlers."
app_usecases: "App struct with NewAppParams (fx.In), repos, logger with .With('APP'). Business methods in separate files per domain concept. Orchestrates repos and external services."
repo_ports: "Interfaces per domain concept (AccountRepository, JWTRepository, etc.) using domain types. Compose related interfaces where needed. Never use API types."
repo_adapters: "var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(NewXRepo, fx.As(new(repo.XRepository))))) for interface binding. Concrete repos in separate files."
migration: "//go:embed internal/repo/v0/data/db/migrations at module root. Sequential numbered files: 001_init.up.sql, 002_feature.up.sql."
settings: "Typed Settings struct with nested configs (DBConfig, RedisConfig) using `settings` tags. const.go with ModuleName, ModuleVersion, RedisPrefix. New() reads from file then unmarshals."
---
