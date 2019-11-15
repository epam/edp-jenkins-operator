# EDP Jenkins Operator

## Overview

The Jenkins operator creates, deploys and manages the EDP Jenkins instance on Kubernetes/OpenShift. The Jenkins instance is equipped with the necessary plugins. 

There is an ability to customize the Jenkins instance and to check the changes during the application creation.

## How to Install Jenkins on a Cluster

Before deploying the Jenkins Operator, pay attention to the prerequisites: 

* Make sure that cluster contains edp service account with the edp-deploy role;
* Check that cluster has definitions for the Jenkins, JenkinsScript and JenkinsServiceAccount CR's; 
    
_**Note**: If the security politics on your cluster are enabled, for consistency, check the security context before deploying the Jenkins Operator._ 
    
After the prerequisites are checked, follow the steps below to install the Jenkins Operator:    
* Apply the deployment template that is placed in the *deploy/operator.yaml* file;
* As soon as the Operator is deployed,  apply the Jenkins CR using the template in the *deploy/crds/v2_v1alpha1_jenkins_cr.yaml* file.

---

In order to apply the necessary customization, get acquainted with the following sections:

* [Add Jenkins Slave](documentation/add-jenkins-slave.md) 
* [Add Job Provision](documentation/add-job-provision.md)
* [Code Review for GitLab](documentation/code-review-for-gitlab.md) 