[![codecov](https://codecov.io/gh/epam/edp-jenkins-operator/branch/master/graph/badge.svg?token=7A2P3UFQWN)](https://codecov.io/gh/epam/edp-jenkins-operator)

# Jenkins Operator

| :heavy_exclamation_mark: Please refer to [EDP documentation](https://epam.github.io/edp-install/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Jenkins Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Jenkins Operator creates, deploys and manages the EDP Jenkins instance, which is equipped with the necessary plugins, on Kubernetes and OpenShift. Also, Jenkins operator is responsible for creating Jenkins job's.

There is an ability to customize the Jenkins instance and to check changes during the application creation.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following the [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/) instruction.

## Installation

In order to install the EDP Jenkins Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```
2. Choose available Helm chart version:
    ```bash
    helm search repo epamedp/jenkins-operator -l
    ```
   Example response:
   ```
     NAME                      CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/jenkins-operator  2.13.1          2.13.1          A Helm chart for EDP Jenkins Operator
     epamedp/jenkins-operator  2.13.0          2.13.0          A Helm chart for EDP Jenkins Operator
     epamedp/jenkins-operator  2.12.1          2.12.1          A Helm chart for EDP Jenkins Operator
     epamedp/jenkins-operator  2.12.0          2.12.0          A Helm chart for EDP Jenkins Operator
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

_**NOTE:** You can add any number of shared libraries. For correct passing values you have to adjust it by index [i]:_

   ```bash
   --set jenkins.sharedLibraries[0].name="stub-name" --set jenkins.sharedLibraries[0].url="stub-url" --set jenkins.sharedLibraries[0].tag="stub-tag" --set jenkins.sharedLibraries[0].secret="stub-secret-name" --set jenkins.sharedLibraries[0].type="ssh"
   ```

_**NOTE:** Adding private shared-library requires pre-condition before deploy - created secret with credentials to private repository:_

Shared library secrets examples:

SSH secret:
   ```
   kind: Secret
   apiVersion: v1
   metadata:
     name: <jenkins.sharedLibraries[i].secret>
     namespace: <edp-project>
   data:
     id_rsa: private-key
     id_rsa.pub: private-key
     username: username
   type: Opaque
  ```

Password secret:
   ```
   kind: Secret
   apiVersion: v1
   metadata:
     name: <jenkins.sharedLibraries[i].secret>
     namespace: <edp-project>
   data:
     first_name: first-name
     last_name: last-name
     password: password
     username: username
   type: Opaque
  ```

Token secret:
   ```
   kind: Secret
   apiVersion: v1
   metadata:
     name: <jenkins.sharedLibraries[i].secret>
     namespace: <edp-project>
   data:
     password: token
     username: username
   type: Opaque
  ```
>_**NOTE**: Due to the unstable work of the Jenkins plugin with the "token" secret type, it is recommended to use the "password" secret type. Simply add the token into the Password field._

4. Install operator in the <edp-project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install jenkins-operator epamedp/jenkins-operator --version <chart_version> --namespace <edp-project> --set name=jenkins-operator --set global.edpName=<edp-project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard>
    ```
5. Check the <edp-project> namespace that should contain operator deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](https://epam.github.io/edp-install/developer-guide/local-development/) page.

Development versions are also available, please refer to the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

* [Architecture Scheme of Jenkins Operator](documentation/arch.md)
* [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/)
