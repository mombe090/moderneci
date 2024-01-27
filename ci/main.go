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
	"os/exec"
	"strings"
	"time"

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
	hcpCloudVaultSecretLoader("ad832508-eed1-4732-9dbc-ea7dd3c363f6", "c37cc2a3-545a-4f30-8578-360572a26a98", "dagger-env")
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
		From("maven:3.9-eclipse-temurin-17").
		WithMountedCache("~/.m2", mavenCache).
		WithMountedCache("/root/.m2", mavenCache).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app")

	// test, scan CVE, build and package application as JAR
	mavenBuilder := app.WithExec([]string{"mvn", "clean", "install", "sonar:sonar", "-Dsonar.projectKey=app-maven", "-Dsonar.host.url=https://sonarcloud.io", "-Dsonar.token=" + os.Getenv("SONAR_TOKEN")})

	// copy JAR files from builderj
	deploy := client.Container().
		From("eclipse-temurin:17-alpine").
		WithDirectory("/app", mavenBuilder.Directory("./target")).
		WithEntrypoint([]string{"java", "-jar", "/app/app.jar"})


	
	// publish image to registry
	address, err := deploy.WithRegistryAuth("docker.io", username, password).
		Publish(ctx, fmt.Sprintf("%s/myapp:%s", username, getImagesTag()))
	if err != nil {
		panic(err)
	} 
	
	// print image address
	fmt.Println("Image published at:", address)

	client.Container().
		From("anchore/grype:latest").
		WithExec([]string{address, "--fail-on", "critical"}).
		Stdout(ctx)

	signImage(ctx, client, source, password, username, address)
	
}

func signImage(ctx context.Context, client *dagger.Client, source *dagger.Directory, password *dagger.Secret, username string, address string) {
	cosignKey := client.SetSecret("cosign-key", os.Getenv("COSIGN_KEY"))
	cosignKeyPassword := client.SetSecret("cosign-key-password", os.Getenv("COSIGN_PASSWORD"))

	cosignContainer := client.Container().
		From("golang:1.21.6").
		WithDirectory("/app", source).
		WithMountedCache("/go/pkg/mod", client.CacheVolume("go-mod-121")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", client.CacheVolume("go-build-121")).
		WithEnvVariable("GOCACHE", "/go/build-cache")

	cosignContainer.
		WithSecretVariable("COSIGN_PASSWORD", cosignKeyPassword).
		WithSecretVariable("COSIGN_KEY", cosignKey).
		WithSecretVariable("REGISTRY_PASSWORD", password).
		WithExec([]string{"sh", "-c", fmt.Sprintf(`
			export GPG_TTY=$(tty)
			git clone https://github.com/sigstore/cosign.git
			cd cosign
			echo $COSIGN_KEY | base64 -d > cosign.key
			go install ./cmd/cosign
			echo $REGISTRY_PASSWORD | $(go env GOPATH)/bin/cosign login --username %s --password-stdin docker.io
			$(go env GOPATH)/bin/cosign sign --key cosign.key %s -y
			exit_status=$?
			if [ $exit_status -eq 0 ]; then
				echo "Command succeeded"
			else
				echo "Command failed with exit status $exit_status"
			fi
			`, username, address)}).
		Stderr(ctx)
}

func hcpCloudVaultSecretLoader(hcpOrgID string, hcpProjID string, appName string) {
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

func getImagesTag() string {
	// Get current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	branch := strings.TrimSpace(string(output))

	//year_month_day_hour_minute_second
	

	timestamp := time.Now().Format("2006_01_02_15_04_05")

	// Generate image tag
	imageTag := fmt.Sprintf("%s_%s", branch, timestamp)

	return imageTag
}