module github.com/riffpad/riffpad/apps/relay

go 1.25.0

require (
	github.com/glebarez/sqlite v1.11.0
	github.com/gorilla/websocket v1.5.3
	github.com/riffpad/riffpad/packages/protocol v0.0.0
	github.com/riffpad/riffpad/packages/webui v0.0.0
	golang.org/x/crypto v0.49.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.25.12
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)

replace github.com/riffpad/riffpad/packages/protocol => ../../packages/protocol

replace github.com/riffpad/riffpad/packages/webui => ../../packages/webui
