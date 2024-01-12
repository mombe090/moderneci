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
		Exclude: []string{"ci"},
	})

	// use maven:3.9 container
	// mount cache and source code volumes
	// set working directory
	app := client.Container().
		From("maven:3.9-eclipse-temurin-17").
		WithMountedCache("~/.m2", mavenCache).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app")

	// test, scan CVE, build and package application as JAR
	build := app.WithExec([]string{"mvn", "clean", "install"})

	// use eclipse alpine container as base
	// copy JAR files from builder
	deploy := client.Container().
		From("eclipse-temurin:17-alpine").
		WithDirectory("/app", build.Directory("./target")).
		WithEntrypoint([]string{"java", "-jar", "/app/app.jar"})

	// publish image to registry
	fmt.Println("Pusblishing image to Docker Hub")
	_, err = deploy.WithRegistryAuth("docker.io", username, password).
		Publish(ctx, fmt.Sprintf("%s/app-maven", username))
	if err != nil {
		panic(err)
	}
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
