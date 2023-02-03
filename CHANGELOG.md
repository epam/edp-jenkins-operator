<a name="unreleased"></a>
## [Unreleased]


<a name="v2.12.2"></a>
## [v2.12.2] - 2023-01-03
### Features

- Custom rest api url for jenkins type [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)

### Routine

- Bump version to 2.12.0 [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Align maven,gradle image version [EPMDEDP-10755](https://jiraeu.epam.com/browse/EPMDEDP-10755)


<a name="v2.13.2"></a>
## [v2.13.2] - 2023-01-23
### Features

- Custom rest api url for jenkins type [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)


<a name="v2.13.1"></a>
## [v2.13.1] - 2022-12-16
### Routine

- Bump edp-library-stages version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)


<a name="v2.13.0"></a>
## [v2.13.0] - 2022-12-13
### Features

- Added a stub linter [EPMDEDP-10536](https://jiraeu.epam.com/browse/EPMDEDP-10536)

### Bug Fixes

- Remove prometheus dependency [EPMDEDP-11049](https://jiraeu.epam.com/browse/EPMDEDP-11049)
- Remove kubernetes-incubator dependency [EPMDEDP-11173](https://jiraeu.epam.com/browse/EPMDEDP-11173)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Change the idleMinutes parameter of npm jenkins agent from 60 to 5 [EPMDEDP-10485](https://jiraeu.epam.com/browse/EPMDEDP-10485)
- Update resource requests and limits for the python jenkins agent [EPMDEDP-10486](https://jiraeu.epam.com/browse/EPMDEDP-10486)
- Bump go agent version [EPMDEDP-10570](https://jiraeu.epam.com/browse/EPMDEDP-10570)
- Bump version to 2.13.0 [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Update jenkins-go-agent to 3.0.15 [EPMDEDP-10735](https://jiraeu.epam.com/browse/EPMDEDP-10735)
- Align maven,gradle image version [EPMDEDP-10755](https://jiraeu.epam.com/browse/EPMDEDP-10755)
- Update current development version [EPMDEDP-10805](https://jiraeu.epam.com/browse/EPMDEDP-10805)
- Bump Jenkins Java8 agents versions [EPMDEDP-11005](https://jiraeu.epam.com/browse/EPMDEDP-11005)
- Remove deprecated Dotnet 2.1 support [EPMDEDP-11024](https://jiraeu.epam.com/browse/EPMDEDP-11024)
- Update jenkins agents version [EPMDEDP-11089](https://jiraeu.epam.com/browse/EPMDEDP-11089)


<a name="v2.12.1"></a>
## [v2.12.1] - 2022-10-28
### Routine

- Align maven,gradle image version [EPMDEDP-10755](https://jiraeu.epam.com/browse/EPMDEDP-10755)


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-08-25
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10085](https://jiraeu.epam.com/browse/EPMDEDP-10085)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Add a new SAST stage into the CI provisioners of kubernetes and openshift platforms [EPMDEDP-10234](https://jiraeu.epam.com/browse/EPMDEDP-10234)
- Add Kubernetes and GitOps libraries stages to job provisioners [EPMDEDP-8257](https://jiraeu.epam.com/browse/EPMDEDP-8257)
- Switch all CRDs to V1 [EPMDEDP-8987](https://jiraeu.epam.com/browse/EPMDEDP-8987)

### Bug Fixes

- Make sure jenkins agents can update codebase status [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Fix typo in ci job-provisioner for OpenShift [EPMDEDP-10131](https://jiraeu.epam.com/browse/EPMDEDP-10131)
- Switch shared library controller to namespace scope instead cluster scope [EPMDEDP-8396](https://jiraeu.epam.com/browse/EPMDEDP-8396)
- Set nullable and optional fields for CRDs [EPMDEDP-8987](https://jiraeu.epam.com/browse/EPMDEDP-8987)

### Code Refactoring

- Deprecate jenkinsJobProvisionBuildNumber value [EPMDEDP-10019](https://jiraeu.epam.com/browse/EPMDEDP-10019)
- Define deploy-templates folder structure [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Deprecate unused Spec components for Jenkins v1 [EPMDEDP-10119](https://jiraeu.epam.com/browse/EPMDEDP-10119)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)
- Define requests and limits for Jenkins and agents [EPMDEDP-10427](https://jiraeu.epam.com/browse/EPMDEDP-10427)

### Routine

- Refactor RBAC [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Update agent images to latest [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Bump helm agent version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Bump version to 2.12.0 [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update Jenkins agents versions [EPMDEDP-10276](https://jiraeu.epam.com/browse/EPMDEDP-10276)
- Update edp-jenkins-go-agent to 3.0.10 [EPMDEDP-10279](https://jiraeu.epam.com/browse/EPMDEDP-10279)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Use deployments as default deploymentType for OpenShift [EPMDEDP-10344](https://jiraeu.epam.com/browse/EPMDEDP-10344)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Add platformType into the openshift and kubernetes job-provisioners [EPMDEDP-10393](https://jiraeu.epam.com/browse/EPMDEDP-10393)
- Align the CI job-provisioner for Kubernetes platform [EPMDEDP-10393](https://jiraeu.epam.com/browse/EPMDEDP-10393)
- Remove extra comma from list of stages [EPMDEDP-10394](https://jiraeu.epam.com/browse/EPMDEDP-10394)
- Remove Kubernetes and GitOps libraries stages from job provisioners [EPMDEDP-10397](https://jiraeu.epam.com/browse/EPMDEDP-10397)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update agent images to latest [EPMDEDP-10414](https://jiraeu.epam.com/browse/EPMDEDP-10414)
- Define requests and limits for Jenkins dotnet agents [EPMDEDP-10427](https://jiraeu.epam.com/browse/EPMDEDP-10427)
- Update chart annotation [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Change container name for jenkins [EPMDEDP-9199](https://jiraeu.epam.com/browse/EPMDEDP-9199)
- Update npm agent image version [EPMDEDP-9243](https://jiraeu.epam.com/browse/EPMDEDP-9243)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v2.11.1"></a>
## [v2.11.1] - 2022-06-30
### Routine

- Update edp-library-stages tag [EPMDEDP-10158](https://jiraeu.epam.com/browse/EPMDEDP-10158)
- Backport Makefile from master branch [EPMDEDP-10158](https://jiraeu.epam.com/browse/EPMDEDP-10158)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-25
### Features

- Discard old builds for CD pipelines [EPMDEDP-8181](https://jiraeu.epam.com/browse/EPMDEDP-8181)
- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Use tags list for the CODEBASE_VERSION for auto deploy. [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Enable stages to provide manual and auto deploy input generation [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Add Kaniko library stages to job provisioners [EPMDEDP-8341](https://jiraeu.epam.com/browse/EPMDEDP-8341)
- Add kaniko-docker agent for Container library [EPMDEDP-8341](https://jiraeu.epam.com/browse/EPMDEDP-8341)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- External shared libraries with custom resource [EPMDEDP-8396](https://jiraeu.epam.com/browse/EPMDEDP-8396)
- Add build pipeline for autotests [EPMDEDP-8920](https://jiraeu.epam.com/browse/EPMDEDP-8920)

### Bug Fixes

- Fix changelog breaking change section [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix GH Actions for release pipeline [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Fix build dockerfile issue [EPMDEDP-8238](https://jiraeu.epam.com/browse/EPMDEDP-8238)
- Switch shared library controller to namespace scope instead cluster scope [EPMDEDP-8396](https://jiraeu.epam.com/browse/EPMDEDP-8396)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- Correct image version [EPMDEDP-8471](https://jiraeu.epam.com/browse/EPMDEDP-8471)
- Jenkins assign role [EPMDEDP-8867](https://jiraeu.epam.com/browse/EPMDEDP-8867)

### Code Refactoring

- Kubernetes provisioner must contain valid content [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- CI Job provisioner must runs on specific Jenkins label [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- Job provisioner must runs on specific Jenkins label [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- Remove duplicate platform mock [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Define namespace for Service Account in Role Binding [EPMDEDP-8084](https://jiraeu.epam.com/browse/EPMDEDP-8084)
- Add timeout for kaniko build step [EPMDEDP-8308](https://jiraeu.epam.com/browse/EPMDEDP-8308)
- Move kaniko provision logic to edp-install [EPMDEDP-8474](https://jiraeu.epam.com/browse/EPMDEDP-8474)

### Testing

- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)

### Routine

- Update Ingress resources to the newest API version [EPMDEDP-7476](https://jiraeu.epam.com/browse/EPMDEDP-7476)
- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Bump Jenkins go agent version [EPMDEDP-7974](https://jiraeu.epam.com/browse/EPMDEDP-7974)
- Bump Jenkins edp-helm agent version [EPMDEDP-7988](https://jiraeu.epam.com/browse/EPMDEDP-7988)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update jenkins URL baseline link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Bump Jenkins helm agent version [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update release flow [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update helm agent image version [EPMDEDP-8257](https://jiraeu.epam.com/browse/EPMDEDP-8257)
- Exclude autogenerated code in SonarQube check. [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Update edp-jenkins-helm-agent version to use helm-docs [EPMDEDP-8329](https://jiraeu.epam.com/browse/EPMDEDP-8329)
- Remove unused parameter from cd provisioner [EPMDEDP-8584](https://jiraeu.epam.com/browse/EPMDEDP-8584)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Bump version to 2.11.0 [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)
- Update npm agent image version [EPMDEDP-9243](https://jiraeu.epam.com/browse/EPMDEDP-9243)

### BREAKING CHANGE:


Custom resource will have two keys: 'tag' for single tag and 'tags' for the list of tags.


<a name="v2.10.1"></a>
## [v2.10.1] - 2022-01-21
### Routine

- Bump Jenkins go agent version [EPMDEDP-7974](https://jiraeu.epam.com/browse/EPMDEDP-7974)
- Update jenkins image and edp-library-stages [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Bump Jenkins helm agent version [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update release flow [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-07
### Features

- Job provisioner is responsible for the formation of Jenkinsfile [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Enable retention for job-provisions builds [EPMDEDP-7439](https://jiraeu.epam.com/browse/EPMDEDP-7439)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- Enable groovy sandbox flag on Openshift [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Fix job-provisioner typo [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Provide Jenkins deploy through deployments on OKD cluster [EPMDEDP-7178](https://jiraeu.epam.com/browse/EPMDEDP-7178)
- Update DotNet-21 jenkins agent version [EPMDEDP-7281](https://jiraeu.epam.com/browse/EPMDEDP-7281)
- Update dotnet jenkins agents version. [EPMDEDP-7281](https://jiraeu.epam.com/browse/EPMDEDP-7281)
- Use Default branch for new branch provisioning [EPMDEDP-7552](https://jiraeu.epam.com/browse/EPMDEDP-7552)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Skip certificate check for Openshift cluster [EPMDEDP-7919](https://jiraeu.epam.com/browse/EPMDEDP-7919)
- Skip certificate check for Openshift cluster [EPMDEDP-7919](https://jiraeu.epam.com/browse/EPMDEDP-7919)

### Code Refactoring

- Replace cluster-wide role/rolebinding to namespaced, remove unused roles [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Rename stage promote-images-ecr to promote-images [EPMDEDP-7378](https://jiraeu.epam.com/browse/EPMDEDP-7378)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Formatting

- Remove spaces and add explicit names [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Routine

- Update openssh-client version [EPMDEDP-7439](https://jiraeu.epam.com/browse/EPMDEDP-7439)
- Bump version to 2.10.0 [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Remove KUBERNETES_MASTER,KUBERNETES_TRUST_CERTIFICATES parameters [EPMDEDP-7879](https://jiraeu.epam.com/browse/EPMDEDP-7879)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update Jenkins agents tags [EPMDEDP-7891](https://jiraeu.epam.com/browse/EPMDEDP-7891)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update operators to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update keycloak to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)

### BREAKING CHANGE:


Job provisioner create jenkinsfile and configure in jenkins pipeline as pipeline script.


<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.3"></a>
## [v2.8.3] - 2021-12-03

<a name="v2.8.2"></a>
## [v2.8.2] - 2021-12-03

<a name="v2.8.1"></a>
## [v2.8.1] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.6"></a>
## [v2.7.6] - 2021-12-03

<a name="v2.7.5"></a>
## [v2.7.5] - 2021-12-03

<a name="v2.7.4"></a>
## [v2.7.4] - 2021-12-03

<a name="v2.7.3"></a>
## [v2.7.3] - 2021-12-03

<a name="v2.7.2"></a>
## [v2.7.2] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03
### Reverts

- [EPMDEDP-5352]: Add CRD access to jenkins sa
- [EPMDEDP-4822] Implement kubernetes Helm install


[Unreleased]: https://github.com/epam/edp-jenkins-operator/compare/v2.12.2...HEAD
[v2.12.2]: https://github.com/epam/edp-jenkins-operator/compare/v2.13.2...v2.12.2
[v2.13.2]: https://github.com/epam/edp-jenkins-operator/compare/v2.13.1...v2.13.2
[v2.13.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.13.0...v2.13.1
[v2.13.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.12.1...v2.13.0
[v2.12.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.12.0...v2.12.1
[v2.12.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.11.1...v2.12.0
[v2.11.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.11.0...v2.11.1
[v2.11.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.10.1...v2.11.0
[v2.10.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.10.0...v2.10.1
[v2.10.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.8.3...v2.9.0
[v2.8.3]: https://github.com/epam/edp-jenkins-operator/compare/v2.8.2...v2.8.3
[v2.8.2]: https://github.com/epam/edp-jenkins-operator/compare/v2.8.1...v2.8.2
[v2.8.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.6...v2.8.0
[v2.7.6]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.5...v2.7.6
[v2.7.5]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.4...v2.7.5
[v2.7.4]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.3...v2.7.4
[v2.7.3]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.2...v2.7.3
[v2.7.2]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-jenkins-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-jenkins-operator/compare/v2.3.0-130...v2.7.0
