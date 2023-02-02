## Arlon e2e tests


### Setup

- First, we run the `testing/e2e_setup.sh` script, which helps us setup a kind management cluster, a git-server [gitea](https://gitea.io/en-us) based workspace repository and installs services like argocd, arlon, capi in the management cluster. It also installs other required tools like kubectl, docker, kind, kuttl, clusterctl, helm and gitea.
  
- In addition to this, the script also creates a capi-eks cluster manifest which serves as the cluster template manifest. This cluster is created as a part of the e2e tests and is pushed to the workspace repository created in the previous step.
  
- This script also adds a  xenial bundle manifest to the workspace repository which is required for creating a xenial bundle and a corresponding profile which is consumed by the test.

- Any prerequisites for any e2e test must be added to this script or to a seperate script as a part of the setup for the e2e test. These scripts must be executed as a part of the e2e test setup before any additional arlon commands can be executed.


### e2e integration tests using KUTTL

- We are using [KUTTL](https://kuttl.dev/) to write e2e integration tests for arlon.

- KUTTL is a declarative integration testing harness for testing operators, KUDO, Helm charts, and any other Kubernetes applications or controllers.

- The KUTTL test CLI organizes tests into suites:
  - A "test step" defines a set of Kubernetes manifests to apply and a state to assert on (wait for or expect).
  - A "test assert" are the part of a test step that define the state to wait for Kubernetes to reach
  
- All the e2e tests are placed in `/testing/e2e` in the respecitive directory for the given test. For example, the test `00-deploy` is used to deploy a cluster and all the files related to this test are placed in `/testing/e2e/00-deploy`. To add a new e2e test here, create a new directory with the correct step index. The tests in `/testing/e2e` run in the order of the index specified.

- Each filename in the test case directory should start with an index (in this example 00) that indicates which test step the file is a part of. The first filename in the test case will begin with index 00, the second will have index 01 and so on. e.g In `/testing/e2e/00-deploy` we have `00-prepare.yaml`, `01-validate.yaml` etc.

- Do note that the tests will run in the order of the index specified. This is the case for both the test case directory and the individual test files within these directories. Files that do not start with a step index are ignored and can be used for documentation or other test data.

- As a part of the test step, we can run commands and execute scripts. Here, for arlon e2e tests, we execute the e2e_setup script as a part of the test step and then run the arlon commands specific to the test case.

- Once we have created a test step case, we can also create a test assert for a given filename. The assert's filename should be the test step index followed by `-assert.yaml` e.g. In `/testing/e2e/00-deploy` we have `02-assert.yaml`, which will run after `02-deploy.yaml` test step.

- In a test assert step, we look for the desired resource state. This is the state of the resource after the test step has finished. As a part of the test assert, we can add a timeout incase the resource that we are waiting on takes a long time to reach the desired state.

- All the test steps and test asserts run in order and each must be successful for the test case to be considered successful. If any test step or test assertion fails then the test will fail.


### Testbed Teardown

- Currently, as a part of the `/testing/e2e_setup_teardown.sh` script, we delete the kind management cluster, the cloned workspace repository, bundles, profiles and cluster manifests present in this repository.

- This script runs at the end of every e2e test run regardless of the success or failure of the test for cleaning up any resources that might have been created as a part of the test.

- Any additional resource that needs to be cleaned up post the e2e test should be added to this script.


### e2e tests integration with Github Actions

- Currently, we are running the arlon e2e tests on an ubuntu VM using Github Actions.

- To invoke the arlon e2e tests, we have a `make` target `make test-e2e` which will be executed by the [e2e test workflow](https://github.com/arlonproj/arlon/blob/main/.github/workflows/e2e.yaml) on Github Actions.

### Running Arlon Tests Locally

- To run the arlon e2e tests locally on your developer setup (Linux or MacOS), export the following environment variables that are required to download CAPA and to setup the git server.
    - GIT_USER - Dummy github username used by gitea helm chart for setting up test user
    - GIT_PASSWORD - Dummy github password fo this test user
    - GIT_EMAIL - Dummy email value for the test user
    - AWS_ACCESS_KEY_ID - AWS access key
    - AWS_SECRET_ACCESS_KEY - AWS secret access key
    - AWS_REGION - AWS region in which you want to run these tests
    - AWS_SSH_KEY_NAME -  refers to the AWS SSH key name in the region specified by `AWS_REGION`
    - AWS_CONTROL_PLANE_MACHINE_TYPE is the instance type of the control plane machines( e.g. t2.medium, t3.large)
    - AWS_NODE_MACHINE_TYPE is the instance type of the worker node machines( e.g. t2.medium, t3.large)

- After exporting these variables, run the make target `make test-e2e` from the repository root.

- This will run the arlon e2e tests and clean up all the resources created by these tests post the completion of this test. Incase, the teardown is not triggered via this target, manually run this make target: `make e2e-teardown`, which runs the teardown script

- The cleanup of AWS resources is a little unreliable and it is advised to clean all the AWS resources (EC2 Instances, VPC, NAT gateways) manually just to avoid incurring further costs.
