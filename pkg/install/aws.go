package install

import (
	"github.com/spf13/cobra"
	"os"
	"sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/cmd/bootstrap/credentials"
	"sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/cmd/bootstrap/iam"
	credentials2 "sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/credentials"
)

const (
	envRegion          = "AWS_REGION"
	envAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	envSessionToken    = "AWS_SESSION_TOKEN"
	envAWSB64Creds     = "AWS_B64ENCODED_CREDENTIALS"
)

type awsInstaller struct {
}

func (a *awsInstaller) EnsureRequisites() error {
	if _, ok := os.LookupEnv(envRegion); !ok {
		return &ErrBootstrap{
			HardFail: true,
			Message:  "AWS_REGION environment variable not set",
		}
	}
	if _, ok := os.LookupEnv(envAccessKeyID); !ok {
		return &ErrBootstrap{
			HardFail: true,
			Message:  "AWS_ACCESS_KEY_ID environment variable not set",
		}
	}
	if _, ok := os.LookupEnv(envSecretAccessKey); !ok {
		return &ErrBootstrap{
			HardFail: true,
			Message:  "AWS_SECRET_ACCESS_KEY environment variable not set",
		}
	}
	if _, ok := os.LookupEnv(envSessionToken); !ok {
		return &ErrBootstrap{
			HardFail: false,
			Message:  "AWS_SESSION_TOKEN environment variable not set, assuming no MFA has been setup",
		}

	}
	return nil
}

func (a *awsInstaller) Bootstrap() error {
	rootCmd := iam.RootCmd()
	var cloudFormationStackCreateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "create-cloudformation-stack" {
			cloudFormationStackCreateCmd = cmd
			break
		}
	}
	err := cloudFormationStackCreateCmd.RunE(rootCmd, []string{})
	if err != nil {
		return &ErrBootstrap{
			HardFail: false,
			Message:  err.Error(),
		}
	}
	credRootCmd := credentials.RootCmd()
	var encodeAsProfileCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "encode-as-profile" {
			encodeAsProfileCmd = cmd
			break
		}
	}
	if err := encodeAsProfileCmd.RunE(credRootCmd, []string{}); err != nil {
		return err
	}
	region := os.Getenv(envRegion)
	awsCreds, err := credentials2.NewAWSCredentialFromDefaultChain(region)
	if err != nil {
		return err
	}
	out, err := awsCreds.RenderBase64EncodedAWSDefaultProfile()
	if err != nil {
		return err
	}
	if err := os.Setenv(envAWSB64Creds, out); err != nil {
		return err
	}
	return nil
}
