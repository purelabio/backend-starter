package main

import (
	"database/sql"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitranim/goh"
	"github.com/mitranim/try"
	"go.uber.org/zap"
)

/*
This must include ALL global mutable state of the application. Makes it easy to
keep track of things and perform global overrides, which is useful for testing.

Defined as an anonymous type because this should be ALWAYS referenced as a
global and never passed around.
*/
var env = func() (env struct {
	conf           Conf               // conf.go
	log            *zap.SugaredLogger // utils_log.go
	db             *sql.DB            // db.go
	serverListener net.Listener       // server.go
	server         *http.Server       // server.go
	rand           *rand.Rand         // utils_text.go
	fileServer     http.Handler       // server.go
}) {
	try.To(env.conf.Init())
	env.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	env.log = env.conf.TryLogger()
	return
}()

// Doing this in `init`, rather than `main`, affects tests.
func init() {
	// Use UTC.
	time.Local = nil

	goh.ErrHandlerDefault = writeErr

	spew.Config.Indent = PRETTY_PRINT_INDENT
	spew.Config.ContinueOnMethod = true
}

func main() {
	try.To(startServer())
}
