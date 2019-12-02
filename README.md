# EDP Jenkins Operator

## Overview

The Jenkins operator creates, deploys and manages the EDP Jenkins instance on Kubernetes/OpenShift. The Jenkins instance is equipped with the necessary plugins. 

There is an ability to customize the Jenkins instance and to check the changes during the application creation.

### Prerequisites
1. Machine with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed with authorized access to the cluster.
2. Admin space is deployed using instruction of [edp-install](https://github.com/epmd-edp/edp-install#admin-space) repository

### Installation
* Go to the [releases](https://github.com/epmd-edp/jenkins-operator/releases) page of this repository, choose a version, download an archive and unzip it.

_**NOTE:** It is highly recommended to use the latest released version._

* Go to the unzipped directory and apply all files with Custom Resource Definitions

`for file in $(ls crds/*_crd.yaml); do kubectl apply -f $file; done`

* Deploy operator

`kubectl patch -n <edp_cicd_project> -f deploy/operator.yaml --local=true --patch='{"spec":{"template":{"spec":{"containers":[{"image":"epamedp/jenkins-operator:<operator_version>", "name":"jenkins-operator-v2", "env": [{"name":"WATCH_NAMESPACE", "value":"<edp_cicd_project>"}, {"name":"PLATFORM_TYPE","value":"<platform>"}]}]}}}}' -o yaml | kubectl -n <edp_cicd_project> apply -f -`

_** <operator_version> - release version you've chosen_

_** <edp_cicd_project> - a namespace or project(in Opensift case) name which you created following [edp-install instructions](https://github.com/epmd-edp/edp-install#install-edp)_

_** <platform_type> - Can be "kubernetes" or "openshift"_

* Check <edp_cicd_project> namespace. It should contain Deployment with your operator up and running.

---

In order to apply the necessary customization, get acquainted with the following sections:

* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)
* [Code Review for GitLab](documentation/code-review-for-gitlab.md) 