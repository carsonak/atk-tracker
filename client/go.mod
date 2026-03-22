module atk-tracker/client

go 1.22

require (
	atk-tracker/shared/go v0.0.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/mattn/go-sqlite3 v1.14.22
	golang.org/x/sys v0.30.0
)

replace atk-tracker/shared/go => ../shared/go
