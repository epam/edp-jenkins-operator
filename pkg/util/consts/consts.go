package consts

const (
	ImportStrategy = "import"

	//statuses
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"

	JenkinsKind       = "Jenkins"
	StageKind         = "Stage"
	JenkinsFolderKind = "JenkinsFolder"
	CDPipelineKind    = "CDPipeline"

	AuthorizationApiGroup = "rbac.authorization.k8s.io"
	ClusterRoleKind       = "ClusterRole"

	JenkinsServiceAccount         = "jenkins"
	EdpAdminConsoleServiceAccount = "edp-admin-console"

	LibraryCodebase = "library"

	JenkinsDefaultScriptConfigMapKey = "context"
)
