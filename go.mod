module github.com/epmd-edp/jenkins-operator/v2

go 1.12

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

require (
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/epmd-edp/keycloak-operator v1.0.18-alpha-39
	github.com/go-openapi/spec v0.19.2
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/spf13/pflag v1.0.3
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472 // indirect
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 // indirect
	golang.org/x/sys v0.0.0-20190904154756-749cb33beabd // indirect
	google.golang.org/appengine v1.6.2 // indirect
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/controller-runtime v0.1.12
)
