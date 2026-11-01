# Cookbook: Common API patterns

Use these RonyKIT patterns to build APIs. The examples come from production services.

## Table of contents

- [Project structure](#project-structure)
- [Service with handler groups](#service-with-handler-groups)
- [Authentication middleware](#authentication-middleware)
- [Pagination](#pagination)
- [Input validation](#input-validation)
- [Error handling patterns](#error-handling-patterns)
- [Repository pattern](#repository-pattern)
- [Configuration and settings](#configuration-and-settings)
- [Dependency injection](#dependency-injection)
- [Rate limiting](#rate-limiting)
- [Health check](#health-check)
- [Webhooks with custom decoders](#webhooks-with-custom-decoders)
- [CORS and server bootstrap](#cors-and-server-bootstrap)
- [Stub generation for service communication](#stub-generation-for-service-communication)
- [Observability](#observability)
- [Panic recovery](#panic-recovery)
- [Handler relay](#handler-relay-dynamic-reverse-proxy)

## Project structure

Organize production services in layers. The `ronyup` scaffolding tool generates this structure:

```
my-service/
├── go.work
├── cmd/all-in-one/
│   └── main.go                # Entry point
├── feature/users/
│   ├── go.mod
│   ├── module.go              # DI wiring (fx.Module)
│   ├── service.go             # Service registration
│   ├── api/
│   │   ├── service.go         # Route definitions
│   │   └── api_user.go        # Handler implementations
│   ├── internal/
│   │   ├── app/
│   │   │   └── app.go         # Business logic
│   │   ├── domain/
│   │   │   ├── types.go       # Domain entities
│   │   │   └── errors.go      # Domain errors
│   │   ├── repo/
│   │   │   ├── port.go        # Repository interfaces
│   │   │   └── v0/            # Repository implementations (sqlc)
│   │   └── settings/
│   │       └── settings.go    # Module configuration
│   ├── gen/stub/gen.go        # Stub code generator
│   └── stub/                  # Generated client stubs
└── pkg/                       # Shared packages
```

Keep handlers thin. Handlers validate input and call the `app` layer. Put business logic in `internal/app`. Put data access behind interfaces in `internal/repo`.

## Service with handler groups

Use `SetupOptionGroup` to group related endpoints:

```go
type RContext = rony.UnaryCtx[rony.EMPTY, rony.NOP]

type Service struct {
	userApp *app.App
}

func New(userApp *app.App) *Service {
	return &Service{userApp: userApp}
}

func (svc *Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup(
		svc.accountHandlers(),
		svc.profileHandlers(),
	)
}

func (svc *Service) accountHandlers() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup(
		rony.WithUnary(svc.CreateAccount,
			rony.POST("/accounts", rony.UnaryName("CreateAccount")),
		),
		rony.WithUnary(svc.GetAccount,
			rony.GET("/accounts/{id}", rony.UnaryName("GetAccount")),
		),
		rony.WithUnary(svc.ListAccounts,
			rony.GET("/accounts", rony.UnaryName("ListAccounts")),
		),
	)
}

func (svc *Service) profileHandlers() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup(
		rony.WithUnary(svc.GetProfile,
			rony.GET("/profile", rony.UnaryName("GetProfile")),
			rony.UnaryHeader(rony.RequiredHeader("Authorization")),
			rony.UnaryMiddleware(svc.MustAuthorized),
		),
		rony.WithUnary(svc.UpdateProfile,
			rony.PUT("/profile", rony.UnaryName("UpdateProfile")),
			rony.UnaryHeader(rony.RequiredHeader("Authorization")),
			rony.UnaryMiddleware(svc.MustAuthorized),
		),
	)
}
```

Register the handler groups with the server:

```go
rony.Setup(srv, "UserService", rony.EmptyState(), apiService.Desc())
```

## Authentication middleware

Use two middleware functions. Parse the token for each request. Enforce authentication on protected endpoints.

```go
func (svc *Service) CheckAuthToken(ctx *kit.Context) {
	token := ctx.In().GetHdr("Authorization")
	if token == "" {
		return
	}
	token = strings.TrimPrefix(token, "Bearer ")

	claim, err := svc.userApp.VerifyToken(ctx.Context(), token)
	if err != nil {
		ctx.SetStatusCode(401)
		ctx.Out().SetMsg(
			errs.B().Code(errs.Unauthenticated).Msg("INVALID_TOKEN").Err(),
		).Send()
		ctx.StopExecution()
		return
	}

	ctx.Set("ctx.user.claim", claim)
}

func (svc *Service) MustAuthorized(ctx *kit.Context) {
	if ctx.Get("ctx.user.claim") == nil {
		ctx.SetStatusCode(401)
		ctx.Out().SetMsg(
			errs.B().Code(errs.Unauthenticated).Msg("UNAUTHORIZED").Err(),
		).Send()
		ctx.StopExecution()
	}
}
```

Register `CheckAuthToken` at the service level. It runs for each request. Register `MustAuthorized` on each protected endpoint:

```go
func (svc *Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup(
		rony.WithMiddleware[rony.EMPTY, rony.NOP](svc.CheckAuthToken),

		// Public - no auth required
		rony.WithUnary(svc.Login, rony.POST("/auth/login")),

		// Protected - requires auth
		rony.WithUnary(svc.GetProfile,
			rony.GET("/profile"),
			rony.UnaryHeader(rony.RequiredHeader("Authorization")),
			rony.UnaryMiddleware(svc.MustAuthorized),
		),
	)
}
```

Get the claim in a handler:

```go
func getUserClaim(ctx *RContext) domain.UserClaim {
	claim, _ := ctx.Get("ctx.user.claim").(domain.UserClaim)
	return claim
}

func (svc *Service) GetProfile(ctx *RContext, in GetProfileInput) (*GetProfileOutput, error) {
	claim := getUserClaim(ctx)
	profile, err := svc.userApp.GetProfile(ctx.Context(), claim.UserID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("FAILED_TO_GET_PROFILE").Cause(err).Err()
	}
	return &GetProfileOutput{Profile: profile}, nil
}
```

## Pagination

Use the same pagination pattern for all list endpoints:

```go
type ListUsersInput struct {
	Page     int32  `json:"page"     query:"page"`
	PageSize int32  `json:"pageSize" query:"pageSize"`
	Status   string `json:"status"   query:"status"`
}

type ListUsersOutput struct {
	Users []User `json:"users"`
	Total int64  `json:"total"`
}

func (svc *Service) ListUsers(ctx *RContext, in ListUsersInput) (*ListUsersOutput, error) {
	page, pageSize := normalizePagination(in.Page, in.PageSize)

	users, total, err := svc.userApp.ListUsers(ctx.Context(), app.ListUsersInput{
		Skip:   (page - 1) * pageSize,
		Limit:  pageSize,
		Status: in.Status,
	})
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("FAILED_TO_LIST_USERS").Cause(err).Err()
	}

	return &ListUsersOutput{Users: users, Total: total}, nil
}

func normalizePagination(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 250 {
		pageSize = 50
	}
	return page, pageSize
}
```

## Input validation

Validate input in the handler. Return structured errors for invalid input:

```go
type CreateUserInput struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	FullName string `json:"fullName"`
}

type CreateUserOutput struct {
	User User `json:"user"`
}

func (svc *Service) CreateUser(ctx *RContext, in CreateUserInput) (*CreateUserOutput, error) {
	if in.Email == "" && in.Phone == "" {
		return nil, errs.B().
			Code(errs.InvalidArgument).
			Msg("EMAIL_OR_PHONE_REQUIRED").
			Err()
	}

	if in.FullName == "" {
		return nil, errs.B().
			Code(errs.InvalidArgument).
			Msg("FULL_NAME_REQUIRED").
			Err()
	}

	user, err := svc.userApp.CreateUser(ctx.Context(), domain.CreateUserInput{
		Email:    in.Email,
		Phone:    in.Phone,
		FullName: in.FullName,
	})
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("FAILED_TO_CREATE_USER").Cause(err).Err()
	}

	return &CreateUserOutput{User: user}, nil
}
```

Use pointer types for optional fields. Use `swag` tags to document OpenAPI enums:

```go
type UpdateUserInput struct {
	ID       int64   `json:"id"`
	FullName *string `json:"fullName"` // optional - nil keeps the current value
	Status   string  `json:"status" swag:"enum:active,inactive"`
}
```

## Error handling patterns

### Define domain errors

```go
// internal/domain/errors.go

var (
	ErrUserNotFound   = errs.B().Code(errs.NotFound).Msg("USER_NOT_FOUND").Err()
	ErrInvalidToken   = errs.GenWrap(errs.FailedPrecondition, "INVALID_TOKEN")
	ErrDuplicateEmail = errs.B().Code(errs.AlreadyExists).Msg("DUPLICATE_EMAIL").Err()
)
```

`errs.GenWrap` creates an error factory that can wrap an original error:

```go
func (app *App) VerifyToken(ctx context.Context, token string) (*Claims, error) {
	claims, err := app.jwtRepo.Verify(token)
	if err != nil {
		return nil, domain.ErrInvalidToken(err)
	}
	return claims, nil
}
```

### Wrap errors in handlers

Keep handlers thin. Add context to app-layer errors:

```go
func (svc *Service) GetUser(ctx *RContext, in GetUserInput) (*GetUserOutput, error) {
	user, err := svc.userApp.GetUser(ctx.Context(), in.ID)
	if err != nil {
		return nil, errs.B().
			Code(errs.Internal).
			Msg("FAILED_TO_GET_USER").
			Cause(err).
			Err()
	}
	return &GetUserOutput{User: user}, nil
}
```

## Repository pattern

### Define interfaces as ports

```go
// internal/repo/port.go

type UserRepository interface {
	Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	Get(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, skip, limit int32, status string) ([]domain.User, int64, error)
	Update(ctx context.Context, id int64, input domain.UpdateUserInput) (*domain.User, error)
}
```

### Implement ports with sqlc

```go
// internal/repo/v0/user.go

var _ repo.UserRepository = (*UserRepo)(nil) // compile-time check

type UserRepo struct {
	q *db.Queries // sqlc-generated
}

func NewUserRepo(q *db.Queries) *UserRepo {
	return &UserRepo{q: q}
}

func (r *UserRepo) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	row, err := r.q.InsertUser(ctx, db.InsertUserParams{
		Email:    sql.NullString{String: input.Email, Valid: input.Email != ""},
		Phone:    sql.NullString{String: input.Phone, Valid: input.Phone != ""},
		FullName: input.FullName,
	})
	if err != nil {
		return nil, err
	}
	return toUser(row), nil
}

func (r *UserRepo) List(ctx context.Context, skip, limit int32, status string) ([]domain.User, int64, error) {
	rows, err := r.q.ListUsers(ctx, db.ListUsersParams{
		Offset: skip,
		Limit:  limit,
		Status: db.NullUserStatus{UserStatus: db.UserStatus(status), Valid: status != ""},
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountUsers(ctx, db.CountUsersParams{
		Status: db.NullUserStatus{UserStatus: db.UserStatus(status), Valid: status != ""},
	})
	if err != nil {
		return nil, 0, err
	}

	users := make([]domain.User, len(rows))
	for i, row := range rows {
		users[i] = *toUser(row)
	}
	return users, total, nil
}
```

### Connect ports with dependency injection

```go
// internal/repo/v0/adapter.go

var Init = fx.Options(
	fx.Provide(
		fx.Private,
		db.New,
		fx.Annotate(NewUserRepo, fx.As(new(repo.UserRepository))),
		fx.Annotate(NewTokenRepo, fx.As(new(repo.TokenRepository))),
	),
)
```

## Configuration and settings

Define a settings struct for each module. Use the `x/settings` package:

```go
// internal/settings/settings.go

type Settings struct {
	DB    DBConfig    `settings:"db"`
	Redis RedisConfig `settings:"redis"`
	JWT   JWTConfig   `settings:"jwt"`
}

type DBConfig struct {
	Host     string `settings:"host"`
	Port     int    `settings:"port"`
	User     string `settings:"user"`
	Password string `settings:"password"`
	DBName   string `settings:"dbname"`
}

type JWTConfig struct {
	Secret string `settings:"secret"`
	TTL    string `settings:"ttl"`
}

func New(set settings.Settings) (*Settings, error) {
	_ = set.SetFromFile("config", "./internal/settings/")
	s := &Settings{}
	if err := set.Unmarshal(s); err != nil {
		return nil, err
	}
	return s, nil
}
```

Example `config.local.yaml` file:

```yaml
db:
  host: localhost
  port: 5432
  user: postgres
  password: secret
  dbname: myapp

redis:
  addr: localhost:6379

jwt:
  secret: my-secret-key
  ttl: 24h
```

## Dependency injection

RonyKIT projects use `uber/fx` and `x/di` to connect dependencies.

### Module wiring

```go
// module.go

var Module = fx.Module(
	"users",
	v0repo.Init,
	diDatasource,
	fx.Provide(
		settings.New,
		app.New,
		api.New,
	),
)

var diDatasource = fx.Options(
	di.ProvideDBParams[settings.Settings](MigrationFS),
	di.ProvideRedisParams[settings.Settings](),
	datasource.InitDB("", ""),
	datasource.InitRedis("", ""),
)
```

### Self-registering services

```go
// service.go

func init() {
	di.RegisterService[*api.Service](di.RegisterServiceParams{
		Kind:     "service",
		Name:     "users",
		InitFn:   LoadSettings,
		ModuleFn: Module,
	})
}
```

### Module discovery in the main entry point

```go
// cmd/all-in-one/main.go

func main() {
	srv := rony.NewServer(
		rony.Listen(":8080"),
		rony.WithServerName("my-api"),
		rony.WithAPIDocs("/docs"),
		rony.UseScalarUI(),
	)

	for _, m := range di.GetServiceByKind("service") {
		opts = append(opts, m(moduleOpts))
	}

	_ = srv.Run(context.Background(), os.Interrupt)
}
```

## Rate limiting

Apply rate-limiting middleware to an endpoint:

```go
rony.WithUnary(svc.CreateOTP,
	rony.POST("/auth/otp"),
	rony.UnaryMiddleware(
		svc.app.RateLimiterHandler(app.RateLimiterInput{
			Limit:  app.PerHour(10),
			Name:   "CreateOTP",
			Fields: []string{"Phone"},
			ValueFn: []app.LimiterValueFunc{
				func(ctx *kit.Context) string { return ctx.Conn().ClientIP() },
			},
		}),
	),
)
```

## Health check

Add a health check endpoint to each production service:

```go
type HealthInput struct{}

type HealthOutput struct {
    Status string `json:"status"`
}

func Healthz(ctx *rony.SUnaryCtx, in HealthInput) (*HealthOutput, error) {
    return &HealthOutput{Status: "ok"}, nil
}

rony.Setup(srv, "Health", rony.EmptyState(),
    rony.WithUnary(Healthz, rony.GET("/healthz", rony.UnaryName("Healthz"))),
)
```

## Webhooks with custom decoders

Use a custom decoder for webhook callbacks with non-standard content types or signatures:

```go
func webhookDecoder(ctx *kit.Context, in []byte) (kit.Message, error) {
    signature := ctx.In().GetHdr("X-Signature")
    if !verifySignature(in, signature) {
        return nil, errs.B().Code(errs.PermissionDenied).Msg("INVALID_SIGNATURE").Err()
    }

    var msg WebhookPayload
    if err := json.Unmarshal(in, &msg); err != nil {
        return nil, err
    }
    return &msg, nil
}

rony.WithRawUnary(svc.HandleWebhook,
    rony.ALL("/webhooks/{provider}",
        rony.UnaryName("HandleWebhook"),
        rony.UnaryDecoder(webhookDecoder),
    ),
)
```

## CORS and server bootstrap

This example configures a server for production:

```go
srv := rony.NewServer(
	rony.Listen(":8080"),
	rony.WithServerName("MyAPI"),
	rony.WithVersion("v1.0.0"),
	rony.WithAPIDocs("/docs"),
	rony.UseScalarUI(),
	rony.WithTracer(tracekit.B3("my-api")),
	rony.WithCORS(rony.CORSConfig{
		IgnoreEmptyOrigin: true,
	}),
	rony.WithErrorHandler(func(ctx *kit.Context, err error) {
		ctx.SetStatusCode(400)
		ctx.Out().SetMsg(
			errs.B().Cause(err).Code(errs.InvalidArgument).Msg("COULD_NOT_PARSE_PAYLOAD").Err(),
		).Send()
	}),
)
```

## Stub generation for service communication

Use generated client stubs for calls between services. Do not write HTTP clients manually.

### Generate stubs for a service

```go
// gen/stub/gen.go

func main() {
	rony.GenerateStub(
		"userstub", "",
		"../stub",
		stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "userstub"}),
		api.Service{}.Desc(),
	)
}
```

Run the generator with `go generate` or a Makefile target.

### Use stubs in another service

```go
// In the gateway or another service's module.go

var diRemoteServices = fx.Options(
	di.ProvideUserStub[settings.Settings](settings.ModuleName),
	di.ProvideLedgerStub[settings.Settings](settings.ModuleName),
)
```

Inject the stub interface into the app layer:

```go
type App struct {
	userStub userc.IUserStub
}

func (a *App) GetUserProfile(ctx context.Context, userID int64) (*Profile, error) {
	resp, err := a.userStub.GetUser(ctx, userc.GetUserRequest{ID: userID})
	if err != nil {
		return nil, err
	}
	return mapToProfile(resp), nil
}
```

## Observability

### Request logging middleware

```go
func logMiddleware(ctx *kit.Context) {
	span := tracekit.Span(ctx.Context())
	tracekit.Event(span, "request",
		attribute.String("route", ctx.Route()),
	)

	ctx.AddModifier(func(out *kit.Envelope) {
		out.SetHdr("Trace-ID", span.SpanContext().TraceID().String())
	})

	ctx.Next()
}
```

### Server-level tracing

```go
rony.NewServer(
	rony.WithTracer(tracekit.B3("my-api")),
	rony.WithLogger(myLogger),
)
```

## Panic recovery

Use global middleware to recover from handler panics:

```go
func recoverPanicMiddleware(ctx *kit.Context) {
    defer func() {
        if r := recover(); r != nil {
            ctx.SetStatusCode(500)
            ctx.Out().SetMsg(
                errs.B().Code(errs.Internal).Msg("INTERNAL_ERROR").Err(),
            ).Send()
            ctx.StopExecution()
        }
    }()
    ctx.Next()
}

rony.NewServer(
    rony.WithGlobalHandlers(recoverPanicMiddleware),
)
```

## Handler relay (dynamic reverse proxy)

Use `rony.WithRelay` when a handler must forward a raw HTTP or WebSocket request. The handler can authenticate the request and resolve a dynamic upstream first. A Kubernetes port-forward is one example.

Use `rony.WithReverseProxy` for a static gateway-level proxy instead.

```go
func (svc *Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup(
		rony.WithMiddleware[rony.EMPTY, rony.NOP](svc.CheckSessionToken),

		rony.WithRelay(
			svc.RelaySessionAPI,
			rony.ANY("/deploy/v1/sessions/:sessionId/api/{path:*}"),
		),
	)
}

func (svc *Service) RelaySessionAPI(ctx *rony.SRelayCtx) error {
	sessionID := /* from route param with ctx.RESTConn() or app helper */
	target, err := svc.app.ResolveSessionUpstream(ctx.Context(), sessionID)
	if err != nil {
		return err
	}
	return ctx.Relay(target, kit.RelayConfig{
		DropQueryParams: []string{"token"},
	})
}
```

- Use `RelayCtx` only. `UnaryCtx` has no `Relay()` method.
- A successful relay writes the upstream response directly. It does not add a JSON envelope.
- Return an `error` for a relay failure. The relay wrapper sends the standard error envelope.

See MCP resource `knowledge://ronyup/architecture/handler-relay` and `std/gateways/fasthttp/proxy/README.md`.

## Next steps

- [Getting Started](./getting-started.md): Learn core concepts and build a server.
- [ronyup Guide](./ronyup-guide.md): Use the scaffolding CLI and MCP server.
- [Architecture](./architecture.md): Learn about internal components.
- [Advanced: Kit](./advanced-kit.md): Build custom gateways with low-level tools.
