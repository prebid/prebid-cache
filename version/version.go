package version

// Ver holds the version derived from the latest git tag
// Populated using:
//
//	go build -ldflags "-X github.com/prebid/prebid-cache/version.Ver=`git describe --tags`"
//
// Populated automatically at build / releases in the Docker image
var Ver string

// Rev holds binary revision string
// Populated using:
//
//	go build -ldflags "-X github.com/prebid/prebid-cache/version.Rev=`git rev-parse HEAD`"
//
// Populated automatically at build / releases in the Docker image
var Rev string
