pipeline{
    agent any
    tools{
        jdk 'jdk'
        maven 'M3'
    }
    environment {
        SCANNER_HOME=tool 'sonar-server'
    }
    stages {
        stage('Workspace Cleaning'){
            steps{
                cleanWs()
            }
        }
        stage('Checkout from Git'){
            steps{
                git branch: 'main', url: 'https://github.com/mombe090/moderneci.git'
            }
        }
        stage('Build App') {
            steps {
                sh "mvn clean install"
            }
        }
        stage("Sonarqube Analysis"){
            steps{
                withSonarQubeEnv('sonar-server') {
                    sh ''' $SCANNER_HOME/bin/sonar-scanner -Dsonar.projectName=Moderneci \
                    -Dsonar.projectKey=Moderneci  \
                    -Dsonar.sources=./target \
                    '''
                }
            }
        }
        stage("Quality Gate"){
            steps {
                script {
                    waitForQualityGate abortPipeline: false, credentialsId: 'sonar-token'
                }
            }
        }
        stage('Install Dependencies') {
            steps {
                sh "mvn clean install"
            }
        }
    }

}