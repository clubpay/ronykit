module github.com/clubpay/ronykit/std/ee/bus/kafka

go 1.27.0

require (
	github.com/clubpay/ronykit/ee v0.0.1
	github.com/clubpay/ronykit/x/rkit v0.6.2
	github.com/twmb/franz-go v1.22.2
	github.com/twmb/franz-go/pkg/kadm v1.19.0
)

require (
	github.com/goccy/go-reflect v1.2.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/pierrec/lz4/v4 v4.1.33 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../../../ee
