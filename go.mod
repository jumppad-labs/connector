module github.com/shipyard-run/connector

go 1.15

require (
	github.com/go-playground/validator/v10 v10.3.0
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-hclog v0.12.1
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.28.0
	google.golang.org/protobuf v1.24.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.13
	k8s.io/apimachinery v0.17.13
	k8s.io/client-go v0.17.13
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d // indirect
)

replace (
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // pinned to release-branch.go1.13
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190821162956-65e3620a7ae7 // pinned to release-branch.go1.13
	k8s.io/api => k8s.io/api v0.17.13
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.13
)
