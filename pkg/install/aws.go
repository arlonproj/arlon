package install

import (
	"flag"
	"fmt"
	"os"

	"github.com/argoproj/argo-cd/v2/util/cli"
	"sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/cmd"
	credentials2 "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/credentials"
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

	envFeatureGateMachinePool = "EXP_MACHINE_POOL"
)

type awsInstaller struct {
	silence bool
}

func (a *awsInstaller) EnsureRequisites() error {
	requiredEnvs := []struct {
		name     string
		hardFail bool
		msg      string
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
			msg:      fmt.Sprintf("%s environment variable not set. MFA enabled accounts will not work.", envSessionToken),
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
			if !a.silence {
				if !a.recoverOnFail(env.msg) {
					return &ErrBootstrap{
						HardFail: true,
						Message:  fmt.Sprintf("%s environment variable not set", env.name),
					}
				}
			}
		}
	}
	return nil
}

func (a *awsInstaller) Bootstrap() error {
	err := os.Setenv(envFeatureGateMachinePool, "true")
	if err != nil {
		return &ErrBootstrap{
			HardFail: true,
			Message:  "Cannot enable AWS machine pool feature gate. Cannot set environment variable EXP_MACHINE_POOL to true",
		}
	}
	ogArgs := os.Args
	defer func() {
		os.Args = ogArgs
	}()
	os.Args = []string{"clusterawsadm", "bootstrap", "iam", "create-cloudformation-stack"}
	if err := flag.CommandLine.Parse([]string{}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		_, _ = fmt.Fprintln(os.Stderr, "")
		return err
	}
	rootCmd := cmd.RootCmd()
	rootCmd.SilenceUsage = true
	err = rootCmd.Execute()
	if err != nil {
		if !a.recoverOnFail("Error when creating cloud-formation-stack and IAM roles. This may be caused by pre-existing IAM roles and may result in a working installation.") {
			return err
		}
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

func (a *awsInstaller) recoverOnFail(message string) bool {
	if !a.silence {
		m := fmt.Sprintf("%s. Continue?[y/n]", message)
		return cli.AskToProceed(m)
	}
	return true
}
