module github.com/ArthurHlt/ptasks

go 1.13

replace github.com/goccy/go-yaml => github.com/orange-cloudfoundry/go-yaml v1.4.0-fix

require (
	github.com/creack/pty v1.1.9
	github.com/jessevdk/go-flags v1.4.0
	github.com/logrusorgru/aurora v0.0.0-20191116043053-66b7ad493a23
	github.com/mattn/go-isatty v0.0.11
	github.com/mattn/go-shellwords v1.0.10
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e // indirect
)
