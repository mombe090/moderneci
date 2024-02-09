package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
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

type Project struct {
	XMLName    xml.Name `xml:"project"`
	ArtifactId string   `xml:"artifactId"`
}

func main() {
	// initialize Dagger client
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// create a cache volume for Maven downloads
	mavenCache := client.CacheVolume("maven-cache")

	// get reference to source code directory
	source := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{"ci", "argocd", "helmChart", "img"},
	})
	// load HCP Cloud Vault secrets
	hcpCloudVaultSecretLoader("ad832508-eed1-4732-9dbc-ea7dd3c363f6", "c37cc2a3-545a-4f30-8578-360572a26a98", "dagger-env")

	// build the java application
	build := appBuilder(client, mavenCache, source)

	// set registry password as secret for Dagger pipeline
	password := client.SetSecret("password", os.Getenv("REGISTRY_PASSWORD"))
	username := os.Getenv("REGISTRY_USERNAME")

	// publish image to registry
	address := publishImageToRegistry(ctx, build, username, password)

	signImage(ctx, client, source, password, username, address)

	scanImageForVulnCheck(ctx, client, address)
}

func appBuilder(client *dagger.Client, mavenCache *dagger.CacheVolume, source *dagger.Directory) *dagger.Container {
	app := client.Container().
		From("maven:3.9-eclipse-temurin-17").
		WithMountedCache("~/.m2", mavenCache).
		WithMountedCache("/root/.m2", mavenCache).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app")

	mavenBuilder := app.WithExec([]string{"mvn", "clean", "install", "sonar:sonar", "-Dsonar.projectKey=app-maven", "-Dsonar.host.url=https://sonarcloud.io", "-Dsonar.token=" + os.Getenv("SONAR_TOKEN")})

	// copy JAR files from builderj
	return client.Container().
		From("eclipse-temurin:17-alpine").
		WithDirectory("/app", mavenBuilder.Directory("./target")).
		WithEntrypoint([]string{"java", "-jar", "/app/app.jar"})
}

func publishImageToRegistry(ctx context.Context, build *dagger.Container, username string, password *dagger.Secret) string {
	address, err := build.WithRegistryAuth("docker.io", username, password).
		Publish(ctx, fmt.Sprintf("%s/%s:%s", username, getAppName(), getImagesTag()))
	if err != nil {
		panic(err)
	}

	// print image address
	fmt.Println("Image published at:", address)
	return address
}

func signImage(ctx context.Context, client *dagger.Client, source *dagger.Directory, password *dagger.Secret, username string, address string) {
	cosignKey := client.SetSecret("cosign-key", os.Getenv("COSIGN_KEY"))
	cosignKeyPassword := client.SetSecret("cosign-key-password", os.Getenv("COSIGN_PASSWORD"))

	keyGenContainer := client.Container().
		From("alpine:3.19.1").
		WithSecretVariable("COSIGN_KEY", cosignKey).
		WithWorkdir("/key").
		WithExec([]string{"sh", "-c", "echo $COSIGN_KEY | base64 -d > cosign.key"})

	_, err := client.Container().
		From("bitnami/cosign:2.2.3").
		WithSecretVariable("COSIGN_PASSWORD", cosignKeyPassword).
		WithSecretVariable("REGISTRY_PASSWORD", password).
		WithDirectory("/app", keyGenContainer.Directory("/key")).
		WithExec([]string{"login", "docker.io", "-u", username, "--p", os.Getenv("REGISTRY_PASSWORD")}).
		WithExec([]string{"sign", "--key", "/app/cosign.key", address, "-y"}).
		Stdout(ctx)

	if err != nil {
		panic(err)
	}

}

func scanImageForVulnCheck(ctx context.Context, client *dagger.Client, address string) {
	out, err := client.Container().
		From("anchore/grype:latest").
		WithExec([]string{address, "--fail-on", "critical"}).
		Stdout(ctx)

	if err != nil {
		panic(err)
	}

	fmt.Println(out)
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

	body, _ := io.ReadAll(resp.Body)
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

	body, _ = io.ReadAll(resp.Body)

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

	fmt.Println("HCP Cloud Vault secrets loaded ")
}

func getImagesTag() string {
	// Get current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	branch := strings.TrimSpace(string(output))

	timestamp := time.Now().Format("2006_01_02_15_04_05")

	// Generate image tag
	imageTag := fmt.Sprintf("%s_%s", branch, timestamp)

	return imageTag
}

func getAppName() string {
	xmlFile, err := os.Open("pom.xml")
	if err != nil {
		fmt.Println(err)
	}
	defer xmlFile.Close()

	byteValue, _ := io.ReadAll(xmlFile)

	var project Project
	xml.Unmarshal(byteValue, &project)

	fmt.Println("Application Name is : " + project.ArtifactId)
	return project.ArtifactId
}
