module github.com/rkrmr33/template-hub

go 1.16

require (
	github.com/go-openapi/runtime v0.20.0
	github.com/go-swagger/go-swagger v0.28.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/rkrmr33/pkg v0.0.1
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.9.0
	go.uber.org/zap v1.19.0 // indirect
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
	k8s.io/apimachinery v0.22.4
)

replace (
	github.com/golang/protobuf => github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-gateway => github.com/grpc-ecosystem/grpc-gateway v1.16.0
	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.2
)
