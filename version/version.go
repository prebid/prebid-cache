package version

// Ver holds the version derived from the latest git tag
// Populated using:
//    go build -ldflags "-X github.com/prebid/prebid-server/version.Ver=`git describe --tags | sed 's/^v//`"
// Populated automatically at build / releases in the Docker image
var Ver string

// VerUnknown is the version used if Ver has not been set by ldflags.
const VerUnknown = "unknown"
