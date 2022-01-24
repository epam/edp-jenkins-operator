<a name="unreleased"></a>
## [Unreleased]

### Bug Fixes

- Fix changelog breaking change section [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix GH Actions for release pipeline [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)

### Code Refactoring

- Remove duplicate platform mock [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Define namespace for Service Account in Role Binding [EPMDEDP-8084](https://jiraeu.epam.com/browse/EPMDEDP-8084)

### Testing

- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests, mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)
- Add tests and mocks [EPMDEDP-7991](https://jiraeu.epam.com/browse/EPMDEDP-7991)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Bump Jenkins go agent version [EPMDEDP-7974](https://jiraeu.epam.com/browse/EPMDEDP-7974)
- Bump Jenkins edp-helm agent version [EPMDEDP-7988](https://jiraeu.epam.com/browse/EPMDEDP-7988)
- Update jenkins URL baseline link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Update release flow [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Bump Jenkins helm agent version [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)


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


[Unreleased]: https://github.com/epam/edp-jenkins-operator/compare/v2.10.1...HEAD
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
