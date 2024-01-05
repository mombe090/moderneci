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
    - Système: ([Anchore](https://anchore.com/), [Trivy](https://github.com/aquasecurity/trivy), [Clair](https://github.com/quay/clair), [Snyk](https://snyk.io/) etc...)
- [SonarQube](https://www.sonarqube.org/)
- Java 17+ et maven 3.8+:
  - Vue que nous allons utiliser une application spring boot 3.2, il est nécessaire d'avoir java et maven d'installé sur votre machine.
- Go 1.20+:
  - Nous allons utiliser dagger avec le SDK go, il est donc nécessaire d'avoir go 1.20+ d'installé sur votre machine.


## Architecture:
Comming

## Code source:
Vous trouverez le code source de l'application ainsi que de celui de la CI dans le dépôt suivant: A COMPLETER

Le choix de l'application est volontairement simple afin de se concentrer sur la pipeline CI/CD, il s'agit d'une application spring boot qui expose un endpoint REST qui retourne un message PONG sur le path /ping.

## Déroulement:
- Check de la qualité du code source: SonarQube
- Check de CVE des librairies: OWASP Dependency Check
- Définition de la pipeline CI avec Dagger SDK (go) 
  - build
  - test
  - scan de vulnérabilité avec trivy
  - sign with Cosign
  - push to docker registry
- Définition des manifests kubernetes pour le deployment de l'application avec (Helm)
- Définition des manifests ArgoCD pour le déploiement de l'application avec (Helm)
- Monitoring du deployment avec ArgoUI
- Monitoring de la sécurité avec Anchore