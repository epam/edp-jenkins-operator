# Jenkins Operator

Get acquainted with the Jenkins Operator and the installation process as well as the local development, 
and architecture scheme.

## Overview

Jenkins Operator creates, deploys and manages the EDP Jenkins instance, which is equipped with the necessary plugins, on Kubernetes and OpenShift.  
Also, Jenkins operator is responsible for creating Jenkins job's.

There is an ability to customize the Jenkins instance and to check changes during the application creation.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epam/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epam/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

## Installation
In order to install the EDP Jenkins Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
    ```bash
     helm search repo epamedp/jenkins-operator
    ```
   Example response:
   ```
     NAME                      CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/jenkins-operator  v2.4.0                          Helm chart for Golang application/service deplo...
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Deploy operator:

    Full available chart parameters list:
    ```
     - <chart_version>                        # Helm chart version;
     - global.edpName                         # a namespace or a project name (in case of OpenShift);
     - global.platform                        # "openshift" or "kubernetes";
     - global.dnsWildCard                     # a cluster DNS wildcard name;
     - global.openshift.deploymentType        # Wich type of kind will be deployed to Openshift (values: deployments/deploymentConfigs);
     - image.name                             # EDP jenkins-oprator Docker image name. The released image can be found on https://hub.docker.com/r/epamedp/jenkins-operator;
     - image.version                          # EDP jenkins-oprator Docker image tag. The released image can be found on https://hub.docker.com/r/epamedp/jenkins-operator/tags;
     - jenkins.deploy                         # Flag to enable/disable Jenkins deploy;
     - jenkins.image                          # EDP Jenkins Docker image name. Default supported is "epamedp/edp-jenkins";
     - jenkins.version                        # EDP Jenkins Docker image tag. Default supported is "2.4.0";
     - jenkins.initImage                      # Init Docker image for Jenkins deployment. Default is "busybox";
     - jenkins.storage.class                  # Storageclass for Jenkins data volume. Default is "gp2";
     - jenkins.storage.size                   # Jenkins data volume capacity. Default is "10Gi";
     - jenkins.sharedLibraries[i].name        # EDP shared-library name;
     - jenkins.sharedLibraries[i].url         # EDP shared-library repository link;
     - jenkins.sharedLibraries[i].tag         # EDP shared-library repository version;
     - jenkins.sharedLibraries[i].secret      # Name of Kubernetes secret which contains credentials to private repository. Use only if repo is private.;
     - jenkins.sharedLibraries[i].type        # Type of connection to repository (ssh or password);
     - jenkins.imagePullSecrets               # Secrets to pull from private Docker registry;
     - jenkins.basePath                       # Base path for Jenkins URL.
    ```

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
     namespace: <edp_cicd_project>
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
     namespace: <edp_cicd_project>
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
     namespace: <edp_cicd_project>
   data:
     password: token
     username: username
   type: Opaque
  ```
>_**NOTE**: Due to the unstable work of the Jenkins plugin with the "token" secret type, it is recommended to use the "password" secret type. Simply add the token into the Password field._

4. Install operator in the <edp_cicd_project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install jenkins-operator epamedp/jenkins-operator --version <chart_version> --namespace <edp_cicd_project> --set name=jenkins-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard>
    ```
5. Check the <edp_cicd_project> namespace that should contain operator deployment with your operator in a running status.

### Related Articles
* [Architecture Scheme of Jenkins Operator](documentation/arch.md)
* [Local Development](documentation/local-development.md)
* [GitHub Integration](https://github.com/epam/edp-admin-console/blob/master/documentation/github-integration.md#github-integration)
* [GitLab Integration](https://github.com/epam/edp-admin-console/blob/master/documentation/gitlab-integration.md#gitlab-integration)
---
* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)

>_**NOTE**: To get more accurate information on the CI/CD customization, please refer to the [admin-console](https://github.com/epam/edp-admin-console/tree/master#edp-admin-console) repository._
