package internal

// Version is the ronyup build version. It defaults to "dev" for local builds
// and is overridden at release time via:
//
//	-ldflags "-X github.com/clubpay/ronykit/ronyup/internal.Version=vX.Y.Z"
var Version = "dev"
