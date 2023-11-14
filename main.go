package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/sethvargo/go-githubactions"
	"strings"
)

func main() {
	app := githubactions.GetInput("app")
	if app == "" {
		githubactions.Fatalf("App must not be null value")
	}

	environment := githubactions.GetInput("environment")
	if environment == "" {
		githubactions.Fatalf("Environment must not be null value")
	}

	app = "termscomparer"
	environment = "dev"
	if app == "" || environment == "" {
		githubactions.Fatalf("Environment and App must not be null values")
	}

	// hierarchy of paths to search for
	// / (root) --> Bath for common environment variables across all apps and environments
	// /${app} --> Path for common environment variables across all environments for a specific app
	// /${environment} --> Path for environment specific variables across all apps
	// /${environment}/${app} --> Path for environment specific variables for a specific app

	defaultPath := "/common"
	defaultAppPath := fmt.Sprintf("/%s", app)
	environmentSpecificPath := fmt.Sprintf("/%s", environment)
	appSpecificPath := fmt.Sprintf("/%s/%s", environment, app)

	paths := []string{
		defaultPath,
		defaultAppPath,
		environmentSpecificPath,
		appSpecificPath,
	}

	ssmClient := ssm.New(session.Must(session.NewSession()))
	ssmVars := make(map[string]string)

	for _, path := range paths {
		githubactions.Infof("Exporting variables from path: %s", path)
		ssmVars = getPathVariables(ssmClient, path, ssmVars, "")
	}

	for k, v := range ssmVars {
		storeVar(k, v)
	}
}

func getPathVariables(client *ssm.SSM, path string, ssmVars map[string]string, nextToken string) map[string]string {
	input := &ssm.GetParametersByPathInput{
		Path:           &path,
		WithDecryption: aws.Bool(true),
		Recursive:      aws.Bool(false),
	}

	if nextToken != "" {
		input.SetNextToken(nextToken)
	}

	output, err := client.GetParametersByPath(input)

	if err != nil {
		githubactions.Fatalf("Error fetching variables from path %s: %s", path, err)
	}

	for _, element := range output.Parameters {
		envKey := *element.Name
		envValue := *element.Value
		envKey = formatEnvKey(envKey, path)
		envValue = formatEnvValue(envValue)
		ssmVars[envKey] = envValue
	}

	if output.NextToken != nil {
		getPathVariables(client, path, ssmVars, *output.NextToken)
	}
	return ssmVars
}

func storeVar(k, v string) {
	githubactions.SetOutput(k, v)
}
func formatEnvKey(in string, path string) string {
	return strings.Replace(strings.Trim(in[len(path):], "/"), "/", "_", -1)

}
func formatEnvValue(in string) string {
	out := strings.Replace(in, "\n", "\\n", -1)
	// Masking in case there is a secret in the variables
	githubactions.AddMask(out)
	return out
}
