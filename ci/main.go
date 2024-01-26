package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"dagger.io/dagger"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type Secret struct {
	Name    string `json:"name"`
	Version struct {
		Value string `json:"value"`
	} `json:"version"`
}

type Response struct {
	Secrets []Secret `json:"secrets"`
}

func main() {
	hcpCloudVaultSecretLoader()
	fmt.Println("HCP Cloud Vault secrets loaded ")

	// initialize Dagger client
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// set registry password as secret for Dagger pipeline
	password := client.SetSecret("password", os.Getenv("REGISTRY_PASSWORD"))
	username := os.Getenv("REGISTRY_USERNAME")

	// create a cache volume for Maven downloads
	mavenCache := client.CacheVolume("maven-cache")

	// get reference to source code directory
	source := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{"ci", "argocd", "helmChart", "img"},
	})

	// use maven:3.9 container
	// mount cache and source code volumes
	// set working directory
	app := client.Container().
		From("maven:3.9.6-amazoncorretto-21-al2023").
		WithMountedCache("~/.m2", mavenCache).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app")

	// test, scan CVE, build and package application as JAR
	mavenBuilder := app.WithExec([]string{"mvn", "clean", "verify", "sonar:sonar"}).
		WithExec([]string{"mvn", "clean", "install"})

	// copy JAR files from builder
	imageBuilder := client.Container().
		From("amazoncorretto:17.0.9").
		WithDirectory("/app", mavenBuilder.Directory("./target")).
		WithEntrypoint([]string{"java", "-jar", "/app/app.jar", "--spring.config.location=file:/app/config/application.yaml"})

	// create a container to install Grype
	/*grypeInstaller := client.Container().
		From("golang:1.21.6").
		WithExec([]string{"go", "install", "github.com/anchore/grype"})

	// scan the image with Grype
	/*grypeRun, err := grypeInstaller.WithExec([]string{"grype", "--fail-on", "critical", "--scope", "image", "--input", "docker://localhost:5000/mombe090/app-maven:1.0.1"})

	// parse the output to count the number of critical vulnerabilities
	criticalCount := strings.Count(scanOutput, "Critical")*/

	// if the number of critical vulnerabilities is more than 5, fail the process
	/*if criticalCount > 5 {
		panic("More than 5 critical vulnerabilities found")
	}*/

	addr, err := imageBuilder.WithRegistryAuth("localhost:5000", username, password).
		Publish(ctx, fmt.Sprintf("localhost:5000/mombe090/app-maven:1.0.1"))

	fmt.Println("Published at:", addr)

	// ...
}

func hcpCloudVaultSecretLoader() {
	data := map[string]string{
		"audience":      "https://api.hashicorp.cloud",
		"grant_type":    "client_credentials",
		"client_id":     os.Getenv("HCP_CLIENT_ID"),
		"client_secret": os.Getenv("HCP_CLIENT_SECRET"),
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post("https://auth.hashicorp.com/oauth/token", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to get HCP API Token: %v", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	tokenResponse := TokenResponse{}
	json.Unmarshal(body, &tokenResponse)

	hcpOrgID := os.Getenv("HCP_ORG_ID")
	hcpProjID := os.Getenv("HCP_PROJ_ID")
	appName := os.Getenv("APP_NAME")

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloud.hashicorp.com/secrets/2023-06-13/organizations/%s/projects/%s/apps/%s/open", hcpOrgID, hcpProjID, appName), nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+tokenResponse.AccessToken)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ = ioutil.ReadAll(resp.Body)

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
	}

	for _, secret := range response.Secrets {
		err = os.Setenv(secret.Name, secret.Version.Value)
		if err != nil {
			log.Fatalf("Failed to set environment variable: %v", err)
		}
	}
}
