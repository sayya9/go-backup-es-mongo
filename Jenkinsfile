#!groovy

def m = env.JOB_NAME =~ "(.*)/(.*)"
def projName = m[0][1]
m = null
parent = "parent-${projName}-${env.BUILD_NUMBER}"

podTemplate(label: parent, containers: []) {
    node(parent) {
        ansiColor('xterm') {
            def jenkinsFile = fileLoader.fromGit('jenkinsfile.groovy', 'git@gitlab.com:your_name/jenkinsfile.git', 'master', 'gitlab-inu', parent)
        }
    }
}
