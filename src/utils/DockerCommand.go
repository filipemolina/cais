package utils

import "os/exec"

// dockerCommand and dockerCommandContext are how every docker invocation in
// this package is built. They are variables rather than direct calls to
// exec.Command so a test can point them at a fake binary and exercise the code
// that reads docker's output without docker being installed - which is most of
// what this package does, and which is why ~21 tests currently need a docker
// on PATH to say anything at all.
//
// The alternative was an interface threaded through nine call sites. It buys
// nothing here: there is one implementation in production, and the thing worth
// testing is the parsing, not the dispatch.
//
// The context form is separate because exactly one call needs cancellation:
// the log stream runs until the user closes the modal rather than until the
// command exits.
//
// A test that swaps either must restore it, and must not run in parallel with
// another that swaps it. That is the price of the seam being this cheap.
var (
	dockerCommand        = exec.Command
	dockerCommandContext = exec.CommandContext
)
