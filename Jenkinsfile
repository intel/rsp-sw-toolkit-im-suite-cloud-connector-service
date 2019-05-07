rrpBuildGoCode {
    projectKey = 'cloud-connector-service'
    testDependencies = ['mongo']
    
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    buildImage = 'amr-registry.caas.intel.com/rrp/ci-go-build-image:1.12.0-alpine'
    dockerImageName = "rsp/${projectKey}"
    protexProjectName = 'bb-cloud-connector-service'

    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']

    infra = [
        stackName: 'RSP-Codepipeline-CloudConnectorService'
    ]
    
    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
