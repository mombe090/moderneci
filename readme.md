# Implémentation d'un processus de CI/CD avec DevSecOps

Ce document montre comment monter une pipeline CI/CD moderne en utilisant les méthodes DevSecOps et Gitops.

Une pipeline CI/CD est un ensemble d'outils et de processus qui permettent de livrer du code de manière automatique et continue (Les outils import peu à ce stade).

- La méthode DevSecOps est une méthode qui permet d'intégrer la sécurité dès le processus de CI jusqu'à la mise en production.
- La méthode Gitops permet de gérer le cycle de vie d'une infrastructure ou d'une appliacation en  se basant sur les fonctionnalités de git (pull-request/merge-request) ainsi que d'un démons qui monitore les branches git concernés puis les appliques vers la cible (clusters/serveurs etc...).

## Prérequis:

- le client [Git](https://git-scm.com/downloads)
- Un editeur de code (VSCode, [Intellij](https://www.jetbrains.com/idea/download/) etc...)
- Un outil de gestion de code source ([gitlab](https://about.gitlab.com/), [github](https://github.com/), [bitbucket](https://bitbucket.org/) etc... public ou on-premise)
- Un compte sur [DockerHub](https://hub.docker.com/), [Harbor](https://goharbor.io/) ou un registre privé ([AWS ECR](https://aws.amazon.com/ecr/), [Azure ACR](https://azure.microsoft.com/en-us/services/container-registry/), [GCP GCR](https://cloud.google.com/container-registry) etc...)
- Un gestionnaire de conteneur ([Docker](https://www.docker.com/), [Podman](https://podman.io/), [Buildah](https://buildah.io/) etc...)
- Un gestionnaire de cluster Kubernetes en local ([rancher-desktop](https://rancher.com/products/rancher-desktop/), [minikube](https://minikube.sigs.k8s.io/docs/start/), [kind](https://kind.sigs.k8s.io/), [k3s](https://k3s.io/) etc...)
- [Hashicorp Vault](https://www.vaultproject.io/):
  - Vault est un outil de gestion de secrets (certificats, mots de passe, tokens etc...) open source développé par Hashicorp.
  - Il permet de stocker et de gérer les secrets de manière sécurisée et centralisée.
  - Vous pouvez l'installer sur votre machine en suivant ce [tutoriel](https://learn.hashicorp.com/tutorials/vault/getting-started-install?in=vault/getting-started) ou bien utiliser Hashicorp Cloud Platform (HCP) qui est une offre cloud de Hashicorp qui permet de déployer et de gérer les produits Hashicorp (Vault, Consul, Nomad, Terraform) en mode SaaS.
- [Cosign](https://docs.sigstore.dev/signing/quickstart/):
  - Est un outis qui permet de signer et de vérifier les images de conteneurs (Docker/OCI) pour s'assurer de l'identité de l'image et de son intégrité avant de la déployer dans un environnement de production.
- [Dagger Engine](https://dagger.io/)
  - Dagger est un moteur de pipeline CI/CD écrit en go, il permet de définir des pipelines en utilisant soit un langage de programmation via un SDK (go, python, javascript) ou en utilisant graphQl.
  - Dagger a été choisi pour sa portabilité et sa flexibilité qui permet d'avoir une même logique pour notre CI quelque soit le vendor (github actions, jenkins, gitlab-ci, tekton ou même sur bash directement).
  - Son avantage est qu'il est possible de définir des pipelines en utilisant un langage de programmation (go, python, javascript) ou en utilisant graphQl et tirer profit de la puissance de ces langages que le yaml ne permet pas mais aussi et surtout déviter le jour on change de vendor de CI/CD de devoir tout réécrire.
  - Voir ce article sur dagger ***A COMPLETER***
  - Il a été dévéloppé par les anciens créateurs de docker (Solomon Hykes et son équipe) et promet d'être la nouvelle génération de pipeline CI/CD comme docker l'a été pour les conteneurs.
- [SonarQube](https://www.sonarqube.org/)
- Un outil de scan de vulnérabilité :
    - librairies applicatives ([OWASP Dependency Check](https://owasp.org/www-project-dependency-check/), [Snyk](https://snyk.io/) etc...)
    - Système: ([Anchore-Grype](https://anchore.com/), [Trivy](https://github.com/aquasecurity/trivy), [Clair](https://github.com/quay/clair), [Snyk](https://snyk.io/) etc...)
- [SonarQube](https://www.sonarqube.org/)
- Java 17+ et maven 3.8+:
  - Vue que nous allons utiliser une application spring boot 3.2, il est nécessaire d'avoir java et maven d'installé sur votre machine.
- Go 1.20+:
  - Nous allons utiliser dagger avec le SDK go, il est donc nécessaire d'avoir go 1.20+ d'installé sur votre machine.


## Architecture:
![Architecture](./img/cicd-arch.png)

## Code source:
Vous trouverez le code source de l'application ainsi que de celui de la CI dans le dépôt suivant : https://github.com/mombe090/moderneci.git

Le choix de l'application est volontairement simple afin de se concentrer sur la pipeline CI/CD, il s'agit d'une application spring boot qui expose un endpoint REST qui retourne un message PONG sur le path /ping.

## Déroulement:
- Check de la qualité du code source : 
  Pour valider la qualité du code et aider les développeurs à améliorer leur code, nous allons utiliser les outils:
  - SonrLint : (plugin VSCode ou intellij) qui permet aux devs de voir les erreurs et les warnings directement sur son IDE.
  - SonarQube : un serveur qui permet de stocker les résultats des analyses de code et de les visualiser ainsi que de fail la pipeline si la qualité du code ne respecte pas les règles définies.
    - Il y a un fichier docker-compose.yml qui permet de déployer SonarQube en local à la ricine du projet.
- Vérification des vulnérabilités connues dans les librairies utilisées par l'application :
  - OWASP Dependency Check : un outil qui permet de scanner les librairies utilisées par l'application.
- Définition de la pipeline CI avec Dagger SDK (go) en local:
  - build :
    - Pour le build de l'application ainsi que la génération des artefactes, nous allons utiliser l'outil Dagger avec son SDK golang.
    - Dans le dossier ci du projet, vous trouverez le fichier main.go qui contient la définition de la pipeline CI.
    - Code: 
    ```go
        package main

        import (
          "context"
          "fmt"
          "log"
          "os"

	      "dagger.io/dagger"
        )

        func main() {
	        vars := []string{"DOCKERHUB_USERNAME", "DOCKERHUB_PASSWORD"}
	        for _, v := range vars {
              if os.Getenv(v) == "" {
                log.Fatalf("Environment variable %s is not set", v)
		      }
	        }

            ctx := context.Background()
            client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
            if err != nil {
                panic(err)
            }
            defer client.Close()
    
            password := client.SetSecret("password", os.Getenv("DOCKERHUB_PASSWORD"))
            username := os.Getenv("DOCKERHUB_USERNAME")
    
            mavenCache := client.CacheVolume("maven-cache")
    
            source := client.Host().Directory(".", dagger.HostDirectoryOpts{
              Exclude: []string{"ci"},
            })
    
            app := client.Container().
                From("maven:3.9-eclipse-temurin-17").
                WithMountedCache("~/.m2", mavenCache).
                WithMountedDirectory("/app", source).
                WithWorkdir("/app")
    
            build := app.WithExec([]string{"mvn", "clean", "install"})
    
            deploy := client.Container().
                From("eclipse-temurin:17-alpine").
                WithDirectory("/app", build.Directory("./target")).
                WithEntrypoint([]string{"java", "-jar", "/app/app.jar"})
    
            address, err := deploy.WithRegistryAuth("docker.io", username, password).
              Publish(ctx, fmt.Sprintf("%s/app-maven", username))
            if err != nil {
              panic(err)
            }
    
            fmt.Println("Image published at:", address)
        }
    ```
  - Dans le code ci-dessus, nous définissons les étapes suivantes :
    - Nous chargeons les variables d'environnement REGYSTRY_USERNAME et REGYSTRY_PASSWORD qui sont utilisées pour se connecter à dockerhub en utilisant Hashicorp Cloud Platform (Vauls Secrets) [HCP](https://portal.cloud.hashicorp.com/).
      - voir la méthode `hcpCloudVaultSecretLoader` qui s'authentifie, récupère les crédentials puis définis à son tour deux variables d'environnements.
      - Ce tutoriel montre comment interagir avec HCP vault secrets : https://developer.hashicorp.com/vault/tutorials/hcp-vault-secrets-get-started/hcp-vault-secrets-retrieve-secret
      - Si vous avez déjà une instance de vault vous pouvez l'utiliser au lieu de la version cloud.
    - Définition d'un cache maven pour éviter de télécharger les dépendances à chaque build en précisant le volume à monter dans le container sera le dossier ~/.m2.
    - On initialise le client dagger pour les actions suivantes :
      - Définition d'un conteneur maven :
        - tester l'application avec les tests unitaires et intégrations.
        - scanner les vulnérabilités connues dans les librairies utilisées par l'application avec OWASP Dependency Check.
        - scanner la qualité du code avec SonarQube.
        - builder l'application et générer les artefacts dont le fichier jar exécutable de spring boot.
      - Définition d'un nouveau conteneur avec jdk qui sera utiliser pour build l'image d'exécution de l'application.
        - On récupère le fichier jar du conteneur de build précedant build et le monte dans le dossier /app du conteneur jdk.
        - On définit le point d'entrée de l'image comme étant le fichier jar de l'application.
        - On publie l'image sur dockerhub avec nos crédentiales.
      - Signature de l'image avec Cosign. A COMPLETER
  - scan de vulnérabilité avec Grype
- Installation d'un cluster kubernetes avec Rancher Desktop/Minikube/Kind/K3s
  - Pour le déploiment de l'application, nous allons utiliser un cluster kubernetes en local.
  - Il existe plusieurs solutions aujourd'hui, personnellement j'utilise Rancher Desktop qui est très performant et permet de choisir facilement les configs de la VM, version de k8s etc avec un UI très explicite.
- Définition des manifests kubernetes pour le deployment de l'application avec (Helm)
- Définition des manifests ArgoCD pour le déploiement de l'application avec (Helm)
- Monitoring du deployment avec ArgoUI
- Monitoring de la sécurité avec Anchore