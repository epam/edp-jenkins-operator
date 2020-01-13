# EDP Jenkins Operator

## Overview

Jenkins operator creates, deploys and manages the EDP Jenkins instance, which is equipped with the necessary plugins, on Kubernetes and OpenShift.  

There is an ability to customize the Jenkins instance and to check changes during the application creation.

### Prerequisites
1. Machine with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed with an authorized access to the cluster;
2. Admin space is deployed by following the repository instruction: [edp-install](https://github.com/epmd-edp/edp-install#admin-space).

### Installation
* Go to the [releases](https://github.com/epmd-edp/jenkins-operator/releases) page of this repository, choose a version, download an archive, and unzip it;

_**NOTE:** It is highly recommended to use the latest released version._

* Go to the unzipped directory and apply all files with the Custom Resource Definitions resource:

`for file in $(ls crds/*_crd.yaml); do kubectl apply -f $file; done`

* Deploy operator:

`kubectl patch -n <edp_cicd_project> -f deploy/operator.yaml --local=true --patch='{"spec":{"template":{"spec":{"containers":[{"image":"epamedp/jenkins-operator:<operator_version>", "name":"jenkins-operator-v2", "env": [{"name":"WATCH_NAMESPACE", "value":"<edp_cicd_project>"}, {"name":"PLATFORM_TYPE","value":"<platform>"}]}]}}}}' -o yaml | kubectl -n <edp_cicd_project> apply -f -`

- _<operator_version> - a selected release version;_

- _<edp_cicd_project> - a namespace or a project name (in case of OpenSift) that is created by following the [edp-install](https://github.com/epmd-edp/edp-install#install-edp) instructions;_

- _<platform_type> - a platform type that can be "kubernetes" or "openshift"_.

* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status.

---

In order to apply the necessary customization, get acquainted with the following sections:

* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)
* [GitLab Integration](documentation/gitlab-integration.md)
* [GitHub Integration](documentation/github-integration.md)
* [Customize CD Pipeline](documentation/customize-deploy-pipeline.md) 
