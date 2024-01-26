verify:
	mvn clean verify -X sonar:sonar -Dsonar.projectKey=Moderneci \
        -Dsonar.projectName='Moderneci' \
        -Dsonar.host.url=http://10.0.0.80:9000  \
        -Dsonar.token=squ_16e782b01810fa18b904ce364aca2a6df3697238

 