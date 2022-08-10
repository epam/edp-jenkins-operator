# jenkins-operator

![Version: 2.12.0-SNAPSHOT](https://img.shields.io/badge/Version-2.12.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.12.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.12.0--SNAPSHOT-informational?style=flat-square)

A Helm chart for EDP Jenkins Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-jenkins-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| global.dnsWildCard | string | `"example.com"` |  |
| global.edpName | string | `""` |  |
| global.openshift.deploymentType | string | `"deployments"` |  |
| global.platform | string | `"openshift"` |  |
| image.repository | string | `"epamedp/jenkins-operator"` |  |
| image.tag | string | `nil` |  |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| jenkins.affinity | object | `{}` |  |
| jenkins.annotations | object | `{}` |  |
| jenkins.deploy | bool | `true` |  |
| jenkins.image | string | `"epamedp/edp-jenkins"` |  |
| jenkins.imagePullPolicy | string | `"IfNotPresent"` |  |
| jenkins.imagePullSecrets | string | `nil` |  |
| jenkins.ingress.annotations | object | `{}` |  |
| jenkins.ingress.pathType | string | `"Prefix"` |  |
| jenkins.ingress.tls | list | `[]` |  |
| jenkins.initImage | string | `"busybox:1.35.0"` |  |
| jenkins.nodeSelector | object | `{}` |  |
| jenkins.resources.limits.memory | string | `"3Gi"` |  |
| jenkins.resources.requests.cpu | string | `"100m"` |  |
| jenkins.resources.requests.memory | string | `"512Mi"` |  |
| jenkins.sharedLibraries[0].name | string | `"edp-library-stages"` |  |
| jenkins.sharedLibraries[0].tag | string | `"master"` |  |
| jenkins.sharedLibraries[0].url | string | `"https://github.com/epam/edp-library-stages.git"` |  |
| jenkins.sharedLibraries[1].name | string | `"edp-library-pipelines"` |  |
| jenkins.sharedLibraries[1].tag | string | `"master"` |  |
| jenkins.sharedLibraries[1].url | string | `"https://github.com/epam/edp-library-pipelines.git"` |  |
| jenkins.storage.class | string | `"gp2"` |  |
| jenkins.storage.size | string | `"10Gi"` |  |
| jenkins.tolerations | list | `[]` |  |
| jenkins.version | string | `"2.12.0-SNAPSHOT"` |  |
| name | string | `"jenkins-operator"` |  |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

