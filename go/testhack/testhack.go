/*
Workaround for the fact that `go test`, unlike `go run`, sets the CWD to the
folder it's testing. This hack allows to run `go test ./go` from the repository
root by "fixing" the CWD for tests.

To use, import this in one of the test files:

	import _ "app/go/testhack"

Doing this in an imported package, rather than in `TestMain`, ensures that the
CWD is changed before running variable initialization and `init` functions in
the root scope of the main package, many of which use the CWD.
*/

package testhack

import "os"

func init() {
	err := os.Chdir("..")
	if err != nil {
		panic(err)
	}
}
