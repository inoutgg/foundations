// Package debug provides utilities useful for debugging.
//
// CAUTION: once this package is imported, you need to specify certain tags
// to the build command to control the debug behavior.
//
// Use `noassert` tag to disable assertion.
// Use `debug` tag to enable debug logging.
// When `production` tag is provided, assertion and debug logging are forcefully
// eliminated.
//
// NOTE: that when debug tooling is disabled it has no runtime penalty.
package debug
