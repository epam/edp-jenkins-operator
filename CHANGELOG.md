<a name="unreleased"></a>
## [Unreleased]


<a name="v2.15.3"></a>
## v2.15.3 - 2023-11-15
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10085](https://jiraeu.epam.com/browse/EPMDEDP-10085)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Add a new SAST stage into the CI provisioners of kubernetes and openshift platforms [EPMDEDP-10234](https://jiraeu.epam.com/browse/EPMDEDP-10234)
- Added a stub linter [EPMDEDP-10536](https://jiraeu.epam.com/browse/EPMDEDP-10536)
- Custom rest api url for jenkins type [EPMDEDP-11198](https://jiraeu.epam.com/browse/EPMDEDP-11198)
- Provide opportunity to use default cluster storageClassName [EPMDEDP-11230](https://jiraeu.epam.com/browse/EPMDEDP-11230)
- Enable extra truststore for Jenkins [EPMDEDP-11529](https://jiraeu.epam.com/browse/EPMDEDP-11529)
- Add the ability to use additional volumes in helm chart [EPMDEDP-11529](https://jiraeu.epam.com/browse/EPMDEDP-11529)
- Add external url param to jenkins spec [EPMDEDP-11854](https://jiraeu.epam.com/browse/EPMDEDP-11854)
- Job provisioner is responsible for the formation of Jenkinsfile [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Enable retention for job-provisions builds [EPMDEDP-7439](https://jiraeu.epam.com/browse/EPMDEDP-7439)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Discard old builds for CD pipelines [EPMDEDP-8181](https://jiraeu.epam.com/browse/EPMDEDP-8181)
- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Add Kubernetes and GitOps libraries stages to job provisioners [EPMDEDP-8257](https://jiraeu.epam.com/browse/EPMDEDP-8257)
- Use tags list for the CODEBASE_VERSION for auto deploy. [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Enable stages to provide manual and auto deploy input generation [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Add kaniko-docker agent for Container library [EPMDEDP-8341](https://jiraeu.epam.com/browse/EPMDEDP-8341)
- Add Kaniko library stages to job provisioners [EPMDEDP-8341](https://jiraeu.epam.com/browse/EPMDEDP-8341)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- External shared libraries with custom resource [EPMDEDP-8396](https://jiraeu.epam.com/browse/EPMDEDP-8396)
- Add build pipeline for autotests [EPMDEDP-8920](https://jiraeu.epam.com/browse/EPMDEDP-8920)
- Switch all CRDs to V1 [EPMDEDP-8987](https://jiraeu.epam.com/browse/EPMDEDP-8987)

### Bug Fixes

- Make sure jenkins agents can update codebase status [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Fix typo in ci job-provisioner for OpenShift [EPMDEDP-10131](https://jiraeu.epam.com/browse/EPMDEDP-10131)
- Remove prometheus dependency [EPMDEDP-11049](https://jiraeu.epam.com/browse/EPMDEDP-11049)
- Modify escaping in CD provisioner [EPMDEDP-11134](https://jiraeu.epam.com/browse/EPMDEDP-11134)
- Add parenthesis in cd provisioner [EPMDEDP-11134](https://jiraeu.epam.com/browse/EPMDEDP-11134)
- Revert Align Jenkins job  provisioners flow [EPMDEDP-11134](https://jiraeu.epam.com/browse/EPMDEDP-11134)
- Remove kubernetes-incubator dependency [EPMDEDP-11173](https://jiraeu.epam.com/browse/EPMDEDP-11173)
- Running a large Jenkins script causes an error [EPMDEDP-11569](https://jiraeu.epam.com/browse/EPMDEDP-11569)
- Define correct image name [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Fix path to git-chglog binary [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Add required overwrite field to the request [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Do not remove assigned role mappings on operator restart [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Enable groovy sandbox flag on Openshift [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Fix job-provisioner typo [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Provide Jenkins deploy through deployments on OKD cluster [EPMDEDP-7178](https://jiraeu.epam.com/browse/EPMDEDP-7178)
- Update dotnet jenkins agents version. [EPMDEDP-7281](https://jiraeu.epam.com/browse/EPMDEDP-7281)
- Update DotNet-21 jenkins agent version [EPMDEDP-7281](https://jiraeu.epam.com/browse/EPMDEDP-7281)
- Use Default branch for new branch provisioning [EPMDEDP-7552](https://jiraeu.epam.com/browse/EPMDEDP-7552)
- Fix changelog breaking change section [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Skip certificate check for Openshift cluster [EPMDEDP-7919](https://jiraeu.epam.com/browse/EPMDEDP-7919)
- Skip certificate check for Openshift cluster [EPMDEDP-7919](https://jiraeu.epam.com/browse/EPMDEDP-7919)
- Fix GH Actions for release pipeline [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Fix build dockerfile issue [EPMDEDP-8238](https://jiraeu.epam.com/browse/EPMDEDP-8238)
- Switch shared library controller to namespace scope instead cluster scope [EPMDEDP-8396](https://jiraeu.epam.com/browse/EPMDEDP-8396)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- Correct image version [EPMDEDP-8471](https://jiraeu.epam.com/browse/EPMDEDP-8471)
- Jenkins assign role [EPMDEDP-8867](https://jiraeu.epam.com/browse/EPMDEDP-8867)
- Set nullable and optional fields for CRDs [EPMDEDP-8987](https://jiraeu.epam.com/browse/EPMDEDP-8987)

### Code Refactoring

- Deprecate jenkinsJobProvisionBuildNumber value [EPMDEDP-10019](https://jiraeu.epam.com/browse/EPMDEDP-10019)
- Define deploy-templates folder structure [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Deprecate unused Spec components for Jenkins v1 [EPMDEDP-10119](https://jiraeu.epam.com/browse/EPMDEDP-10119)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)
- Define requests and limits for Jenkins and agents [EPMDEDP-10427](https://jiraeu.epam.com/browse/EPMDEDP-10427)
- Fixed golangci-lint warnings [EPMDEDP-10627](https://jiraeu.epam.com/browse/EPMDEDP-10627)
- Kubernetes provisioner must contain valid content [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- CI Job provisioner must runs on specific Jenkins label [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- Job provisioner must runs on specific Jenkins label [EPMDEDP-6800](https://jiraeu.epam.com/browse/EPMDEDP-6800)
- Replace cluster-wide role/rolebinding to namespaced, remove unused roles [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Rename stage promote-images-ecr to promote-images [EPMDEDP-7378](https://jiraeu.epam.com/browse/EPMDEDP-7378)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)
- Remove duplicate platform mock [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Define namespace for Service Account in Role Binding [EPMDEDP-8084](https://jiraeu.epam.com/browse/EPMDEDP-8084)
- Add timeout for kaniko build step [EPMDEDP-8308](https://jiraeu.epam.com/browse/EPMDEDP-8308)
- Move kaniko provision logic to edp-install [EPMDEDP-8474](https://jiraeu.epam.com/browse/EPMDEDP-8474)

### Formatting

- Remove spaces and add explicit names [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Testing

- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)

### Routine

- Refactor RBAC [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Update agent images to latest [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Bump helm agent version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
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
- Change the idleMinutes parameter of npm jenkins agent from 60 to 5 [EPMDEDP-10485](https://jiraeu.epam.com/browse/EPMDEDP-10485)
- Update resource requests and limits for the python jenkins agent [EPMDEDP-10486](https://jiraeu.epam.com/browse/EPMDEDP-10486)
- Bump go agent version [EPMDEDP-10570](https://jiraeu.epam.com/browse/EPMDEDP-10570)
- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Update jenkins-go-agent to 3.0.15 [EPMDEDP-10735](https://jiraeu.epam.com/browse/EPMDEDP-10735)
- Align maven,gradle image version [EPMDEDP-10755](https://jiraeu.epam.com/browse/EPMDEDP-10755)
- Update current development version [EPMDEDP-10805](https://jiraeu.epam.com/browse/EPMDEDP-10805)
- Bump Jenkins Java8 agents versions [EPMDEDP-11005](https://jiraeu.epam.com/browse/EPMDEDP-11005)
- Remove deprecated Dotnet 2.1 support [EPMDEDP-11024](https://jiraeu.epam.com/browse/EPMDEDP-11024)
- Update jenkins agents version [EPMDEDP-11089](https://jiraeu.epam.com/browse/EPMDEDP-11089)
- Align Jenkins job  provisioners flow [EPMDEDP-11134](https://jiraeu.epam.com/browse/EPMDEDP-11134)
- Align Jenkins job provisioners flow [EPMDEDP-11134](https://jiraeu.epam.com/browse/EPMDEDP-11134)
- Update current development version [EPMDEDP-11472](https://jiraeu.epam.com/browse/EPMDEDP-11472)
- Update git-chglog for jenkins-operator [EPMDEDP-11518](https://jiraeu.epam.com/browse/EPMDEDP-11518)
- Change the value of the parameter that contains the name of the secret [EPMDEDP-11529](https://jiraeu.epam.com/browse/EPMDEDP-11529)
- Bump golang.org/x/net from 0.5.0 to 0.8.0 [EPMDEDP-11578](https://jiraeu.epam.com/browse/EPMDEDP-11578)
- Upgrade alpine image version to 3.16.4 [EPMDEDP-11764](https://jiraeu.epam.com/browse/EPMDEDP-11764)
- Bump version to 2.15.0 [EPMDEDP-11826](https://jiraeu.epam.com/browse/EPMDEDP-11826)
- Bump dockerfile packages version [EPMDEDP-11928](https://jiraeu.epam.com/browse/EPMDEDP-11928)
- Add templates for github issues [EPMDEDP-11928](https://jiraeu.epam.com/browse/EPMDEDP-11928)
- Bump sast agent version [EPMDEDP-11949](https://jiraeu.epam.com/browse/EPMDEDP-11949)
- Upgrade alpine image version to 3.18.0 [EPMDEDP-12085](https://jiraeu.epam.com/browse/EPMDEDP-12085)
- Address security issues [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Address CVE-2022-28948 issue [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Update GH release flow [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Update all dependencies [EPMDEDP-12608](https://jiraeu.epam.com/browse/EPMDEDP-12608)
- Update openssh-client version [EPMDEDP-7439](https://jiraeu.epam.com/browse/EPMDEDP-7439)
- Update Ingress resources to the newest API version [EPMDEDP-7476](https://jiraeu.epam.com/browse/EPMDEDP-7476)
- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Remove KUBERNETES_MASTER,KUBERNETES_TRUST_CERTIFICATES parameters [EPMDEDP-7879](https://jiraeu.epam.com/browse/EPMDEDP-7879)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update Jenkins agents tags [EPMDEDP-7891](https://jiraeu.epam.com/browse/EPMDEDP-7891)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update keycloak to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update operators to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Bump Jenkins go agent version [EPMDEDP-7974](https://jiraeu.epam.com/browse/EPMDEDP-7974)
- Bump Jenkins edp-helm agent version [EPMDEDP-7988](https://jiraeu.epam.com/browse/EPMDEDP-7988)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update jenkins URL baseline link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update release flow [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Bump Jenkins helm agent version [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update helm agent image version [EPMDEDP-8257](https://jiraeu.epam.com/browse/EPMDEDP-8257)
- Exclude autogenerated code in SonarQube check. [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Update edp-jenkins-helm-agent version to use helm-docs [EPMDEDP-8329](https://jiraeu.epam.com/browse/EPMDEDP-8329)
- Remove unused parameter from cd provisioner [EPMDEDP-8584](https://jiraeu.epam.com/browse/EPMDEDP-8584)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Update agents image versions [EPMDEDP-8808](https://jiraeu.epam.com/browse/EPMDEDP-8808)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)
- Change container name for jenkins [EPMDEDP-9199](https://jiraeu.epam.com/browse/EPMDEDP-9199)
- Update npm agent image version [EPMDEDP-9243](https://jiraeu.epam.com/browse/EPMDEDP-9243)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update chart and application version in Readme file [EPMDEDP-11221](https://jiraeu.epam.com/browse/EPMDEDP-11221)
- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)

### Reverts

- [EPMDEDP-5352]: Add CRD access to jenkins sa
- [EPMDEDP-4822] Implement kubernetes Helm install

### BREAKING CHANGE:


Custom resource will have two keys: 'tag' for single tag and 'tags' for the list of tags.

Job provisioner create jenkinsfile and configure in jenkins pipeline as pipeline script.


[Unreleased]: https://github.com/epam/edp-jenkins-operator/compare/v2.15.3...HEAD
