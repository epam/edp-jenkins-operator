# EDP Jenkins Operator

## Overview

Jenkins operator creates, deploys and manages the EDP Jenkins instance, which is equipped with the necessary plugins, on Kubernetes and OpenShift.  

There is an ability to customize the Jenkins instance and to check changes during the application creation.

### Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/release-2.3/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/release-2.3/documentation/kubernetes_install_edp.md#edp-namespace).

### Installation
* Go to the [releases](https://github.com/epmd-edp/jenkins-operator/releases) page of this repository, choose a version, download an archive, and unzip it;

_**NOTE:** It is highly recommended to use the latest released version._

* Go to the unzipped directory and deploy operator:
```bash
helm install jenkins-operator --namespace <edp_cicd_project> --set name=jenkins-operator --set namespace=<edp_cicd_project> --set platform=<platform_type> --set image.name=epamedp/jenkins-operator --set image.version=<operator_version> deploy-templates
```

- _<edp_cicd_project> - a namespace or a project name (in case of OpenShift) that is created by one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/release-2.3/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/release-2.3/documentation/kubernetes_install_edp.md#edp-namespace);_ 

- _<platform_type> - a platform type that can be "kubernetes" or "openshift";_

- _<operator_version> - a selected release version tag for the operator from Docker Hub;_

* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status

---

In order to apply the necessary customization, get acquainted with the following sections:

* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)
* [GitLab Integration](documentation/gitlab-integration.md)
* [GitHub Integration](documentation/github-integration.md)
* [Customize CD Pipeline](documentation/customize-deploy-pipeline.md)
* [Local Development](documentation/local-development.md)