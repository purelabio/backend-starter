package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeshaw/envdecode"
	"github.com/joho/godotenv"
	"github.com/mitranim/try"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ENV_FILE_NAME                = ".env.properties"
	FILE_MODE_DEFAULT            = os.FileMode(0600)
	FILE_MODE_WWW                = os.FileMode(0644)
	DIR_MODE_DEFAULT             = os.FileMode(0700)
	CHAN_SIZE_SMALL              = 64
	CHAN_SIZE_LARGE              = 1024
	FEED_SIZE_DEFAULT            = 24
	FEED_SIZE_MAX                = 48
	CTX_DB_TX_KEY                = "db_tx"
	CTX_REQ_KEY                  = "req"
	PRETTY_PRINT_INDENT          = "  "
	LOWERCASE_LETTERS            = "abcdefghijklmnopqrstuvwxyz"
	LOWERCASE_LETTERS_AND_DIGITS = LOWERCASE_LETTERS + "0123456789"
)

var (
	DEFAULT_DB_TX_OPTIONS = &sql.TxOptions{}
)

type Conf struct {
	ServerPort         int           `env:"SERVER_PORT,required"`
	PostgresDbName     string        `env:"POSTGRES_DB_NAME,required"`
	PostgresDbHost     string        `env:"POSTGRES_DB_HOST,required"`
	PostgresDbPort     string        `env:"POSTGRES_DB_PORT"`
	PostgresUser       string        `env:"POSTGRES_USER,required"`
	PostgresPassword   string        `env:"POSTGRES_PASSWORD"`
	PostgresSearchPath string        `env:"POSTGRES_SEARCH_PATH,required"`
	PublicDir          string        `env:"PUBLIC_DIR"`
	LogLevel           zapcore.Level `env:"LOG_LEVEL"`
	LogOutput          string        `env:"LOG_OUTPUT"`
	DevelopmentMode    bool          `env:"DEVELOPMENT_MODE"`
	PrettyJson         bool          `env:"PRETTY_JSON"`
	PrettyXml          bool          `env:"PRETTY_XML"`
	PrettySql          bool          `env:"PRETTY_SQL"`
}

func (self *Conf) Init() error {
	// Using `CONF` allows to alter the config location when invoking the app:
	//
	//   CONF=some_folder go run ./go
	var path = filepath.Join(os.Getenv("CONF"), ENV_FILE_NAME)

	// The file is optional. We use Kubernetes' support for env vars; it defines
	// them in the environment rather than a file.
	err := godotenv.Load(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.WithStack(err)
	}

	err = envdecode.StrictDecode(self)
	return errors.WithStack(err)
}

func (self Conf) PostgresConnString() string {
	vals := []string{
		"host=" + self.PostgresDbHost,
		"dbname=" + self.PostgresDbName,
		"sslmode=disable",
		"user=" + self.PostgresUser,
		"search_path=" + self.PostgresSearchPath,
		"timezone=UTC",
	}
	if self.PostgresPassword != "" {
		vals = append(vals, "password="+self.PostgresPassword)
	}
	if self.PostgresDbPort != "" {
		vals = append(vals, "port="+self.PostgresDbPort)
	}
	return strings.Join(vals, " ")
}

func (self Conf) TryLogger() *zap.SugaredLogger {
	var logConf zap.Config
	if self.DevelopmentMode {
		logConf = zap.NewDevelopmentConfig()
	} else {
		logConf = zap.NewProductionConfig()
	}

	logConf.Level = zap.NewAtomicLevelAt(self.LogLevel)

	if self.LogOutput != "" {
		logConf.OutputPaths = []string{self.LogOutput}
	}

	log, err := logConf.Build()
	try.To(err)
	return log.Sugar()
}
