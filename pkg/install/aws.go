package install

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/cmd/bootstrap/credentials"
	"sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/cmd/bootstrap/iam"
	credentials2 "sigs.k8s.io/cluster-api-provider-aws/cmd/clusterawsadm/credentials"
)

const (
	envRegion           = "AWS_REGION"
	envAccessKeyID      = "AWS_ACCESS_KEY_ID"
	envSecretAccessKey  = "AWS_SECRET_ACCESS_KEY"
	envSessionToken     = "AWS_SESSION_TOKEN"
	envSSHKeyName       = "AWS_SSH_KEY_NAME"
	envCtrlPlaneMachine = "AWS_CONTROL_PLANE_MACHINE_TYPE"
	envNodeMachine      = "AWS_NODE_MACHINE_TYPE"
	envAWSB64Creds      = "AWS_B64ENCODED_CREDENTIALS"
)

type awsInstaller struct {
}

func (a *awsInstaller) EnsureRequisites() error {
	requiredEnvs := []struct {
		name     string
		hardFail bool
	}{
		{
			name:     envRegion,
			hardFail: true,
		},
		{
			name:     envAccessKeyID,
			hardFail: true,
		},
		{
			name:     envSecretAccessKey,
			hardFail: true,
		},
		{
			name:     envSessionToken,
			hardFail: false,
		},
		{
			name:     envSSHKeyName,
			hardFail: true,
		},
		{
			name:     envCtrlPlaneMachine,
			hardFail: true,
		},
		{
			name:     envNodeMachine,
			hardFail: true,
		},
	}
	for _, env := range requiredEnvs {
		if val := os.Getenv(env.name); len(val) == 0 {
			if env.hardFail {
				return &ErrBootstrap{
					HardFail: env.hardFail,
					Message:  fmt.Sprintf("%s environment variable not set", env.name),
				}
			}
			fmt.Printf("%s environment variable not set\n", env.name)
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
