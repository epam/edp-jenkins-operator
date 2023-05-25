# jenkins-operator

![Version: 2.15.0-SNAPSHOT](https://img.shields.io/badge/Version-2.15.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.15.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.15.0--SNAPSHOT-informational?style=flat-square)

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
| extraVolumeMounts | list | `[]` | Additional volumeMounts to be added to the container |
| extraVolumes | list | `[]` | Additional volumes to be added to the pod |
| global.dnsWildCard | string | `nil` | a cluster DNS wildcard name |
| global.edpName | string | `""` | namespace or a project name (in case of OpenShift) |
| global.openshift.deploymentType | string | `"deployments"` | Wich type of kind will be deployed to Openshift (values: deployments/deploymentConfigs) |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/jenkins-operator"` | EDP jenkins-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/jenkins-operator) |
| image.tag | string | `nil` | EDP jenkins-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/jenkins-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| jenkins.affinity | object | `{}` |  |
| jenkins.annotations | object | `{}` |  |
| jenkins.caCerts.enabled | bool | `false` | Flag for enabling additional CA certificates |
| jenkins.caCerts.image | string | `"adoptopenjdk/openjdk11:alpine"` | Change init CA certificates container image |
| jenkins.caCerts.secret | string | `"secret-name"` | Name of the secret containing additional CA certificates |
| jenkins.deploy | bool | `true` | Flag to enable/disable Jenkins deploy |
| jenkins.image | string | `"epamedp/edp-jenkins"` | EDP Jenkins Docker image name. Default supported is "epamedp/edp-jenkins" |
| jenkins.imagePullPolicy | string | `"IfNotPresent"` |  |
| jenkins.imagePullSecrets | string | `nil` | Secrets to pull from private Docker registry |
| jenkins.ingress.annotations | object | `{}` |  |
| jenkins.ingress.pathType | string | `"Prefix"` | pathType is only for k8s >= 1.1= |
| jenkins.ingress.tls | list | `[]` | See https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/#specifying-the-class-of-an-ingress ingressClassName: nginx |
| jenkins.initImage | string | `"busybox:1.35.0"` | Init Docker image for Jenkins deployment. Default is "busybox" |
| jenkins.jenkinsJavaOptions | string | `""` | Values to add to JENKINS_JAVA_OPTIONS |
| jenkins.nodeSelector | object | `{}` |  |
| jenkins.resources.limits.memory | string | `"5Gi"` |  |
| jenkins.resources.requests.cpu | string | `"1000m"` |  |
| jenkins.resources.requests.memory | string | `"1500Mi"` |  |
| jenkins.sharedLibraries[0] | object | `{"name":"edp-library-stages","tag":"v2.16.0","url":"https://github.com/epam/edp-library-stages.git"}` | EDP shared-library name |
| jenkins.sharedLibraries[0].tag | string | `"v2.16.0"` | EDP shared-library repository version |
| jenkins.sharedLibraries[0].url | string | `"https://github.com/epam/edp-library-stages.git"` | EDP shared-library repository link |
| jenkins.sharedLibraries[1].name | string | `"edp-library-pipelines"` |  |
| jenkins.sharedLibraries[1].tag | string | `"v2.16.0"` |  |
| jenkins.sharedLibraries[1].url | string | `"https://github.com/epam/edp-library-pipelines.git"` |  |
| jenkins.storage.size | string | `"10Gi"` | Jenkins data volume capacity |
| jenkins.tolerations | list | `[]` |  |
| jenkins.version | string | `"2.13.0"` | EDP Jenkins Docker image tag |
| name | string | `"jenkins-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

