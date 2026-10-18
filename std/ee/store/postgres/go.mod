module github.com/clubpay/ronykit/std/ee/store/postgres

go 1.27.0

require (
	github.com/clubpay/ronykit/ee v0.0.1
	github.com/clubpay/ronykit/x/rkit v0.6.2
	github.com/jackc/pgx/v5 v5.11.0
)

require (
	github.com/goccy/go-reflect v1.2.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../../../ee
