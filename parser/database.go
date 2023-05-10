package parser

var dbText = `package %s

import (
	"fmt"
	"log"
	"os"
	"time"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func PostgresConnection(dsn string, timezone string, logLevel logger.LogLevel) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			loc, err := time.LoadLocation(timezone)
			if err != nil {
				return time.Now()
			}
			return time.Now().In(loc)
		},
		PrepareStmt:                      true,
		IgnoreRelationshipsWhenMigrating: false,
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel: logLevel,
			}),
	})

	if err != nil {
		return nil, err
	}

	// ping database
	err = ping(db)
	if err != nil {
		return nil, fmt.Errorf("ping(): %w", err)
	}

	// Use a connection pool
	err = setConnPool(db)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ping(db *gorm.DB) error {
	rawConn, err := db.DB()
	if err != nil {
		return err
	}
	return rawConn.Ping()
}

func setConnPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(20)
	return nil
}

// Postgres configuration struct.
// Holds fields when a DSN is parsed from a string.
type DatabaseConfig struct {
	Database string // dbname
	User     string // user
	Password string // password, default ""
	Host     string // host, default: localhost
	Port     string // postgres port, default 5432
	SSLMode  string // ssl_mode, default=disabled
	Timezone string // Timezone
}

// parse postgres DSN into DatabaseConfig struct
// If config is nil, it does nothing.
func ParseDSN(dsn string, config *DatabaseConfig) {
	if config == nil {
		return
	}

	configMap := map[string]string{}
	for _, s := range strings.Split(dsn, " ") {
		v := strings.Split(s, "=")

		if len(v) == 2 {
			configMap[v[0]] = v[1]
		}
	}

	config.Database = configMap["dbname"]
	config.User = configMap["user"]
	config.Host = configMap["host"]
	config.Password = configMap["password"]
	config.Timezone = configMap["TimeZone"]

	if configMap["port"] != "" {
		config.Port = configMap["port"]
	} else {
		config.Port = "5432"
	}

	if configMap["sslmode"] != "" {
		config.SSLMode = configMap["sslmode"]
	} else {
		config.SSLMode = "disabled"
	}
}
`
