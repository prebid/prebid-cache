package endpoints

const (
	MaxValueLength = 1024 * 10
	MaxNumValues   = 10
)

// This status code signals that we're having trouble reaching a dependent service (currently Azure).
// This service sits behind an nginx load balancer which considers the normal 500 and 504 errors to
// be a sign of bad service health. If te service responds with these, it will stop forwarding traffic
// in case the service is dying.
//
// However... we're running behind Kubernetes. The Horizontal Pod Autoscaler should take care of
// "an overwhelmed service" by allocating more machines. If nginx scales back the traffic, the HPA
// scales *down* those machines... and creates a vicious cycle.
//
// Kurt Adam says he's working on a solution for this... but until it's ready, we'll use this
// non-standard 5xx response to dodge nginx if Azure times out.
const HttpDependencyTimeout = 597