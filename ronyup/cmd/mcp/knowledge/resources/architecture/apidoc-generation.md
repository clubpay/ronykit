Generate API documentation through the `rony` server options for the runtime
service, and `x/apidoc` for standalone artifacts.

- On the live server, enable docs with `rony.WithAPIDocs("/docs")` plus one of
  `rony.UseScalarUI()`, `rony.UseSwaggerUI()`, or `rony.UseRedocUI()`.
- For standalone OpenAPI/Postman generation, call `apidoc.New(title, version,
desc)` and serve `Generator.SwaggerUI(svcDesc)` / `ReDocUI` / `ScalarUI` as an
  `fs.FS` from any handler.
- Always regenerate consumer-facing artifacts after contract changes
  (`make gen-stub` for typed clients, `apidoc.New` for OpenAPI specs).
