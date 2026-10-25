module github.com/clubpay/ronykit/std/ee/bus/nats

go 1.27.0

require (
	github.com/clubpay/ronykit/ee v0.0.5
	github.com/clubpay/ronykit/x/rkit v0.6.4
	github.com/nats-io/nats.go v1.54.0
)

require (
	github.com/goccy/go-reflect v1.2.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../../../ee
