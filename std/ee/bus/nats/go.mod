module github.com/clubpay/ronykit/std/ee/bus/nats

go 1.27.0

require (
	github.com/clubpay/ronykit/ee v0.0.0
	github.com/nats-io/nats.go v1.54.0
)

require (
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../../../ee
