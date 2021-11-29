module github.com/cadence-oss/cadence-notification

go 1.17

require (
	github.com/uber-go/tally v3.3.15+incompatible
	github.com/uber/cadence v0.22.3
	github.com/urfave/cli v1.22.4
	github.com/apache/thrift v0.13.0
)

// Pin the thrift version to mitigate issue: https://github.com/uber-go/cadence-client/issues/1129#issuecomment-932554933
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7