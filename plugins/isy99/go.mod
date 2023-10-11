module github.com/iotdomain/isy99

go 1.13

require (
	github.com/iotdomain/iotdomain-go v0.0.0-20200928060533-3e6dc24cf1bb
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1

)

// Temporary for testing iotdomain-go until release
replace github.com/iotdomain/iotdomain-go => ../iotdomain-go
