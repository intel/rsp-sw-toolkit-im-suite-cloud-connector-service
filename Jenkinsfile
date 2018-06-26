rrpBuildGoCode {
    projectKey = 'cloud-connector-service'
    testDependencies = ['mongo']
    
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"

    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']

    infra = [
        stackName: 'RRP-CodePipeline-CloudConnectorService'
    ]
}