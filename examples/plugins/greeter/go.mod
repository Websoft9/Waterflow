module github.com/Websoft9/waterflow/examples/plugins/greeter

go 1.24.0

toolchain go1.24.5

require (
	github.com/Websoft9/waterflow v0.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// 本地开发使用 replace
replace github.com/Websoft9/waterflow => ../../..
