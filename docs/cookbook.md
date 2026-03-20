# Cookbook: Common API Patterns

Production-ready patterns for building APIs with RonyKIT. These examples are drawn from
real-world usage in production services.

## Table of Contents

- [Project Structure](#project-structure)
- [Service with Handler Groups](#service-with-handler-groups)
- [Authentication Middleware](#authentication-middleware)
- [Pagination](#pagination)
- [Input Validation](#input-validation)
- [Error Handling Patterns](#error-handling-patterns)
- [Repository Pattern](#repository-pattern)
- [Configuration and Settings](#configuration-and-settings)
- [Dependency Injection](#dependency-injection)
- [Rate Limiting](#rate-limiting)
- [Health Check](#health-check)
- [Webhooks with Custom Decoders](#webhooks-with-custom-decoders)
- [CORS and Server Bootstrap](#cors-and-server-bootstrap)
- [Stub Generation for Service Communication](#stub-generation-for-service-communication)
- [Observability](#observability)
- [Panic Recovery](#panic-recovery)

---

## Project Structure

For production services, organize your code in layers. The `ronyup` scaffolding tool
generates this structure automatically:

```
my-service/
├── go.work
├── cmd/service/
│   └── main.go                # Entry point
├── feature/service/users/
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

**Key principle:** Handlers are thin — they validate input and delegate to the `app`
layer. Business logic lives in `internal/app`. Data access lives behind interfaces in
`internal/repo`.

---

## Service with Handler Groups

Organize related endpoints into handler groups using `SetupOptionGroup`:

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

Then register with the server:

```go
rony.Setup(srv, "UserService", rony.EmptyState(), apiService.Desc())
```

---

## Authentication Middleware

Use a two-tier approach: parse the token on every request, enforce auth per-endpoint.

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

Register `CheckAuthToken` at service level (runs on every request) and
`MustAuthorized` on individual protected endpoints:

```go
func (svc *Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
    return rony.SetupOptionGroup(
        rony.WithMiddleware[rony.EMPTY, rony.NOP](svc.CheckAuthToken),

        // Public — no auth required
        rony.WithUnary(svc.Login, rony.POST("/auth/login")),

        // Protected — requires auth
        rony.WithUnary(svc.GetProfile,
            rony.GET("/profile"),
            rony.UnaryHeader(rony.RequiredHeader("Authorization")),
            rony.UnaryMiddleware(svc.MustAuthorized),
        ),
    )
}
```

Extract the claim in handlers:

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

---

## Pagination

Define a consistent pagination pattern across all list endpoints:

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

---

## Input Validation

Validate inputs in the handler and return structured errors:

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

Use **pointer types** for truly optional fields and `swag` tags for OpenAPI enum documentation:

```go
type UpdateUserInput struct {
    ID       int64   `json:"id"`
    FullName *string `json:"fullName"`           // optional — nil means "don't update"
    Status   string  `json:"status" swag:"enum:active,inactive"`
}
```

---

## Error Handling Patterns

### Define domain errors

```go
// internal/domain/errors.go

var (
    ErrUserNotFound   = errs.B().Code(errs.NotFound).Msg("USER_NOT_FOUND").Err()
    ErrInvalidToken   = errs.GenWrap(errs.FailedPrecondition, "INVALID_TOKEN")
    ErrDuplicateEmail = errs.B().Code(errs.AlreadyExists).Msg("DUPLICATE_EMAIL").Err()
)
```

`errs.GenWrap` creates a wrappable error factory — useful when you want to include
the original error as context:

```go
func (app *App) VerifyToken(ctx context.Context, token string) (*Claims, error) {
    claims, err := app.jwtRepo.Verify(token)
    if err != nil {
        return nil, domain.ErrInvalidToken(err)
    }
    return claims, nil
}
```

### Handler-level wrapping

Keep handlers thin. Wrap app-layer errors with context:

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

---

## Repository Pattern

### Define interfaces (ports)

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

### Implement with sqlc

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

### Wire with DI

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

---

## Configuration and Settings

Each module defines its own settings struct using the `x/settings` package:

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

Config file (`config.local.yaml`):

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

---

## Dependency Injection

RonyKIT projects use `uber/fx` with the `x/di` package for DI wiring.

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

### Main entry point discovers modules

```go
// cmd/service/main.go

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

---

## Rate Limiting

Apply rate limiting as per-endpoint middleware:

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

---

## Health Check

Every production service should have a health check endpoint:

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

---

## Webhooks with Custom Decoders

For webhook callbacks that use non-standard content types or signatures:

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

---

## CORS and Server Bootstrap

A production-ready server bootstrap:

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

---

## Stub Generation for Service Communication

In a microservices architecture, services communicate via generated client stubs
instead of hand-written HTTP clients.

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

Run with `go generate` or as part of your Makefile.

### Consume stubs in another service

```go
// In the gateway or another service's module.go

var diRemoteServices = fx.Options(
    di.ProvideUserStub[settings.Settings](settings.ModuleName),
    di.ProvideLedgerStub[settings.Settings](settings.ModuleName),
)
```

Then inject the stub interface in your app layer:

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

---

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

---

## Panic Recovery

Protect all handlers from panics:

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

---

## Next Steps

- [Getting Started](./getting-started.md) — core concepts and first server
- [ronyup Guide](./ronyup-guide.md) — scaffolding CLI and MCP server
- [Architecture](./architecture.md) — how RonyKit works internally
- [Advanced: Kit](./advanced-kit.md) — low-level toolkit for custom gateways
