module ex12

go 1.27.2

require (
	github.com/clubpay/ronykit/ee v0.0.0
	github.com/clubpay/ronykit/x/rkit v0.6.1
)

require (
	github.com/goccy/go-reflect v1.2.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../ee
