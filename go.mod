module github.com/epmd-edp/jenkins-operator/v2

go 1.14

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37

require (
	github.com/bndr/gojenkins v0.2.1-0.20181125150310-de43c03cf849
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/epmd-edp/cd-pipeline-operator/v2 v2.3.0-58.0.20200424095303-90cf9714dc73
	github.com/epmd-edp/codebase-operator/v2 v2.3.0-88
	github.com/epmd-edp/edp-component-operator v0.0.0-20191029131047-d87b7a6d3bb0
	github.com/epmd-edp/gerrit-operator/v2 v2.1.0-52
	github.com/epmd-edp/keycloak-operator v1.0.31-alpha-56
	github.com/go-openapi/spec v0.19.3
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.6.0
	github.com/spf13/pflag v1.0.3
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/controller-runtime v0.1.12
)
