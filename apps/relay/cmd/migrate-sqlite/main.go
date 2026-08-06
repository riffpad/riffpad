// Command migrate-sqlite copies relay metadata from a SQLite database into a
// Postgres database. It is a one-way, idempotency-guarded migration: the tool
// refuses to overwrite non-empty target tables unless --force is given.
//
// Usage:
//
//	go run ./apps/relay/cmd/migrate-sqlite \
//	  -sqlite /var/lib/riffpad-relay/relay.db \
//	  -postgres 'postgres://riffpad:PASSWORD@127.0.0.1:5432/riffpad?sslmode=disable'
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/glebarez/sqlite"
	"github.com/riffpad/riffpad/apps/relay/internal/hub"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type tableDef struct {
	model any
}

var tables = []tableDef{
	{model: &hub.User{}},
	{model: &hub.OAuthAccount{}},
	{model: &hub.AuthToken{}},
	{model: &hub.HostRecord{}},
	{model: &hub.Device{}},
	{model: &hub.SessionMeta{}},
}

func copyTable[T any](src, dst *gorm.DB, name string) (int64, error) {
	var rows []T
	if err := src.Table(name).Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("read %s: %w", name, err)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := dst.Table(name).Create(&rows).Error; err != nil {
		return 0, fmt.Errorf("write %s: %w", name, err)
	}
	return int64(len(rows)), nil
}

func main() {
	sqlitePath := flag.String("sqlite", "", "path to the SQLite relay.db")
	dsn := flag.String("postgres", os.Getenv("DATABASE_URL"), "Postgres DSN")
	force := flag.Bool("force", false, "copy even when the target table already has rows")
	flag.Parse()
	if *sqlitePath == "" || *dsn == "" {
		flag.Usage()
		os.Exit(2)
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}
	src, err := gorm.Open(sqlite.Open(*sqlitePath+"?_pragma=busy_timeout(5000)"), gormCfg)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	dst, err := gorm.Open(postgres.Open(*dsn), gormCfg)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}

	for _, t := range tables {
		table := dst.NamingStrategy.TableName(reflect.TypeOf(t.model).Elem().Name())
		if err := dst.AutoMigrate(t.model); err != nil {
			log.Fatalf("migrate %s: %v", table, err)
		}
		var count int64
		if err := dst.Table(table).Count(&count).Error; err != nil {
			log.Fatalf("count %s: %v", table, err)
		}
		if count > 0 && !*force {
			log.Fatalf("%s already has %d rows; pass --force to overwrite", table, count)
		}
	}

	for _, t := range tables {
		table := dst.NamingStrategy.TableName(reflect.TypeOf(t.model).Elem().Name())
		var copied int64
		switch t.model.(type) {
		case *hub.User:
			copied, err = copyTable[hub.User](src, dst, table)
		case *hub.OAuthAccount:
			copied, err = copyTable[hub.OAuthAccount](src, dst, table)
		case *hub.AuthToken:
			copied, err = copyTable[hub.AuthToken](src, dst, table)
		case *hub.HostRecord:
			copied, err = copyTable[hub.HostRecord](src, dst, table)
		case *hub.Device:
			copied, err = copyTable[hub.Device](src, dst, table)
		case *hub.SessionMeta:
			copied, err = copyTable[hub.SessionMeta](src, dst, table)
		default:
			log.Fatalf("no copier for %s", table)
		}
		if err != nil {
			log.Fatalf("copy %s: %v", table, err)
		}
		fmt.Printf("copied %s: %d rows\n", table, copied)
	}
	fmt.Println("migration complete")
}
