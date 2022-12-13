package spec

import "fmt"

const (
	// JenkinsDefaultAdminUser - default Jenkins admin user.
	JenkinsDefaultAdminUser string = "admin"

	// JenkinsDefaultUiPort default port for Jenkins UI in service.
	JenkinsDefaultUiPort = 8080

	EdpAnnotationsPrefix string = "edp.epam.com"

	JenkinsAdminPasswordSuffix string = "admin-password"

	JenkinsTokenAnnotationSuffix string = "admin-token"

	RouteHTTPSScheme = "https"

	RouteHTTPScheme = "http"
)

var (
	Replicas int32 = 1

	TerminationGracePeriod = int64(30)

	Command = []string{"sh", "-c", fmt.Sprintf(
		"JENKINS_HOME=\"/var/lib/jenkins\"; mkdir -p $JENKINS_HOME/.ssh; if [ -d /tmp/ssh ];" +
			"then chmod 777 -R $JENKINS_HOME/.ssh; cat /tmp/ssh/id_rsa > $JENKINS_HOME/.ssh/id_rsa;" +
			"chmod 400 $JENKINS_HOME/.ssh/id_rsa; if [ -e $JENKINS_HOME/.ssh/config ]; " +
			"then chmod 400 -fR $JENKINS_HOME/.ssh/config; fi; fi")}
)
