# Testbed and E2E "smoke test"

## Instructions

- Use a fresh Ubuntu 20.04 or 22.04 system
- Minimum specs: 8 GB RAM, 2 vCPUs, 30 GB disk (the last one is important)
- git clone arlon repo
- cd to repo top directory
- check out `private/leb/testbed` branch (temporary)
- run: `testing/ubuntu_devel_prereqs.sh`
- run: `testing/ubuntu_testbed_prereqs.sh` (this step can be run in parallel in a separate window, a few seconds after starting the previous step)
- log out and log back in to ensure you have the right permissions to run docker (run `docker ps` to verify)
- create testbed: `testing/ensure_testbed.sh`
- optionally run E2E smoke test: `testing/test_basecluster_deploy_with_capd.sh`. This registers a base cluster that uses CAPD, and deploys a workload cluster.
- optional cleanup: `arlon cluster delete capd-1`.
- Unfortunately, there is a bug in CAPD provider that won't clean up some resources. To clean up manually:
  - get a list of remaining dockermachines: `kubectl -n capd-1 get dockermachine`
  - for each of those, delete them by editing it: `kubectl -n capd-1 edit dockermachine xxx` to remove all finalizers. This should cause the resource to go away.
  - verify that all k8s resources got cleaned up: `argocd app list` should eventually produce empty list
  - clean up docker containers: run `docker ps -a` to see all containers. Delete the ones prefixed with `capd-1-` using `docker stop` and `docker rm` (or `docker rm -f`)
- to clean up the entire testbed, run `testing/teardown_testbed.sh`

