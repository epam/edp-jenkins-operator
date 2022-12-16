/* Copyright 2022 EPAM Systems.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and
limitations under the License. */

import groovy.json.*
import jenkins.model.Jenkins
import hudson.model.labels.LabelAtom
import javaposse.jobdsl.plugin.*
import com.cloudbees.hudson.plugins.folder.*

def getProvisioner = new URL('https://raw.githubusercontent.com/epam/edp-jenkins-operator/master/build/configs/job-provisions/ci/default-ci-provisioner').openConnection()
String scriptText = ""
if (getProvisioner.getResponseCode().equals(200)) {
    scriptText = getProvisioner.getInputStream().getText()
}

def jobName = "default"
def folderName = "job-provisions"
def ciFolderName = "ci"
def folder = Jenkins.instance.getItem(folderName)
if (folder == null) {
  folder = Jenkins.instance.createProject(Folder.class, folderName)
}
def ciFolder = folder.getItem(ciFolderName)
if (ciFolder == null) {
  ciFolder = folder.createProject(Folder.class, ciFolderName)
}
def project = ciFolder.getItem(jobName)
if (project == null) {
  project = ciFolder.createProject(FreeStyleProject, jobName)
}
project.getBuildersList().clear()
executeDslScripts = new ExecuteDslScripts()
executeDslScripts.setScriptText(scriptText)
project.setBuildDiscarder(new hudson.tasks.LogRotator (10, 10))
project.getBuildersList().add(executeDslScripts)
def definitionList = [new StringParameterDefinition("NAME", ""),
                      new StringParameterDefinition("TYPE", ""),
                      new StringParameterDefinition("BUILD_TOOL", ""),
                      new StringParameterDefinition("BRANCH", ""),
                      new StringParameterDefinition("GIT_SERVER_CR_NAME", ""),
                      new StringParameterDefinition("GIT_SERVER_CR_VERSION", ""),
                      new StringParameterDefinition("GIT_CREDENTIALS_ID", ""),
                      new StringParameterDefinition("REPOSITORY_PATH", ""),
                      new StringParameterDefinition("JIRA_INTEGRATION_ENABLED", ""),
                      new StringParameterDefinition("PLATFORM_TYPE", ""),
                      new StringParameterDefinition("DEFAULT_BRANCH", "")]

project.addProperty(new ParametersDefinitionProperty(definitionList))
project.setConcurrentBuild(true)
project.setAssignedLabel(new LabelAtom("master"))
project.save()
