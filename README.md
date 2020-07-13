# EDP Jenkins Operator

## Overview

Jenkins operator creates, deploys and manages the EDP Jenkins instance, which is equipped with the necessary plugins, on Kubernetes and OpenShift.  

There is an ability to customize the Jenkins instance and to check changes during the application creation.

### Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

### Installation
In order to install the EDP Jenkins-operator, follow the steps below:

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
    - global.platform                        # "openShift" or "kubernetes";
    - global.dnsWildCard                     # a cluster DNS wildcard name;
    - image.name                             # EDP jenkins-oprator Docker image name. The released image can be found on https://hub.docker.com/r/epamedp/jenkins-operator;
    - image.version                          # EDP jenkins-oprator Docker image tag. The released image can be found on https://hub.docker.com/r/epamedp/jenkins-operator/tags;
    - jenkins.deploy                         # Flag to enable/disable Jenkins deploy;
    - jenkins.image                          # EDP Jenkins Docker image name. Default supported is "epamedp/edp-jenkins";
    - jenkins.version                        # EDP Jenkins Docker image tag. Default supported is "2.4.0";
    - jenkins.initImage                      # Init Docker image for Jenkins deployment. Default is "busybox";
    - jenkins.storage.class                  # Storageclass for Jenkins data volume. Default is "gp2";
    - jenkins.storage.size                   # Jenkins data volume capacity. Default is "10Gi";
    - jenkins.libraryPipelinesRepo           # EDP shared-library-pipelines repository link;
    - jenkins.libraryPipelinesVersion        # EDP shared-library-pipelines repository version;
    - jenkins.libraryStagesRepo              # EDP shared-library-stages repository link;
    - jenkins.libraryStagesVersion           # EDP shared-library-stages repository version;
    - jenkins.imagePullSecrets               # Secrets to pull from private Docker registry;
    - jenkins.basePath                       # Base path for Jenkins URL.
```
Set your parameters and launching a Helm chart deployment. Example command:
```bash
helm install jenkins-operator epamedp/jenkins-operator --version <chart_version> --namespace <edp_cicd_project> --set name=jenkins-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard> --set image.name=epamedp/jenkins-operator --set image.version=<operator_version>
```
* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status

In order to apply the necessary customization, get acquainted with the following sections:

* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)
* [GitLab Integration](documentation/gitlab-integration.md)
* [GitHub Integration](documentation/github-integration.md)
* [Customize CD Pipeline](documentation/customize-deploy-pipeline.md)
* [Local Development](documentation/local-development.md)