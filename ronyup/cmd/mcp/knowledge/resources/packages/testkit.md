---
import_path: github.com/clubpay/ronykit/x/testkit
short_name: testkit
---

Integration-test harness: fx graph + settings + Gnomock containers for Postgres/Redis. Required for every repository port method.

## Usage Hint

```go
func Setup(t *testing.T, populates ...any) {
	t.Helper()
	set := &settings.Settings{ /* test DB/Redis fields */ }
	testkit.Run(
		t,
		fx.Supply(set),
		testkit.InitDB("db", testkit.InitDBParams{
			User: set.DB.User, Pass: set.DB.Pass, DB: set.DB.DB,
			Queries: testkit.FolderContent("../v0/data/db/migrations"),
		}),
		testkit.InitRedis("redis", testkit.InitRedisParams{}),
		v0repo.Init,
		fx.Populate(populates...),
	)
}
```

- `testkit.Run(t, opts ...fx.Option)` builds and tears down the fx app.
- `testkit.InitDB` / `testkit.InitRedis` start Gnomock containers (need Docker).
- Assert with testify (`assert` / `require`). App unit tests use fakes and do **not** need testkit.

Do not roll your own docker-compose / `sql.Open` test harness for repo ports.
