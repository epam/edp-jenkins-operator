package spec

const (
	//JenkinsDefaultAdminUser - default Jenkins admin user
	JenkinsDefaultAdminUser string = "admin"

	// JenkinsDefaultJnlpPort default port for JNLP process to connect in service
	JenkinsDefaultJnlpPort int32 = 50000

	// JenkinsDefaultUiPort default port for Jenkins UI in service
	JenkinsDefaultUiPort = 8080

	//JenkinsDefaultMemoryRequest default value for Jenkins` container memory request
	JenkinsDefaultMemoryRequest string = "500Mi"

	//JenkinsRecreateTimeout default timeout for recreate strategy in DeploymentConfig
	JenkinsRecreateTimeout int64 = 6000

	JenkinsPasswordSecretName string = "admin-password"
)
