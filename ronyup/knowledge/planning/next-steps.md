1. Run create_feature (or execute setup_command) to scaffold files using the standard service template.
2. Implement contracts in api/service.go; keep handlers thin and delegate to app use-cases.
3. Implement business orchestration in internal/app and keep persistence behind internal/repo/port.go interfaces.
4. Default persistence to Postgres and implement repository queries with sqlc unless another storage stack is explicitly requested.
5. If persistence is needed, add adapter implementations under internal/repo/v0 and update migration/settings files.
6. Wire the service via x/di.RegisterService in module.go so it participates in the fx lifecycle and is discoverable by the server.
7. Define a typed settings struct and unmarshal it with x/settings.Unmarshal; provide settings via fx for api and app layers.
8. Inject x/telemetry/logkit.Logger into the app layer and use it for all logging; add x/telemetry/tracekit.W3C() spans for cross-service calls.
9. Generate OpenAPI docs with x/apidoc from your service descriptors and mount the Swagger/ReDoc/Scalar UI endpoint.
10. If the service has user-facing messages, set up x/i18n bundles and use i18n.TextCtx for per-request locale translations.
11. Add in-memory caching with x/cache where read-heavy paths benefit from TTL-based Ristretto partitions.
12. Write integration tests using x/testkit.Run with Gnomock containers; use testkit.InitDB/InitRedis for infra dependencies.
13. Run make gen-stub in the feature module to regenerate stubs after contract changes.
14. Use generated stubs from other services for inter-service calls instead of duplicating client transport code.
15. Run go test ./... and go fmt ./... in the new feature module before committing.
