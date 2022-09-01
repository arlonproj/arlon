# Testbed and E2E "smoke test"

## Instructions

- Use a fresh Ubuntu 20.04 or 22.04 system
- Minimum specs: 8 GB RAM, 2 vCPUs, 30 GB disk (the last one is important)
- git clone arlon repo
- cd to repo top directory
- check out `private/leb/testbed` branch (temporary)
- run: `testing/ubuntu_devel_prereqs.sh`
- run: `testing/ubuntu_testbed_prereqs.sh`
- log out and log back in to ensure you have the right permissions to run docker
- create testbed: `testing/ensure_testbed.sh`
- optionally run E2E smoke test: testing/test_basecluster_deploy_with_capd.sh
- optional cleanup: `arlon cluster delete capd-1`. Unfortunately, there is a bug in CAPD provider that won't clean up the last Docker container (used by control plane)
- to clean up manually the things that CAPD didn't do properly:
  - get a list of remaining dockermachines: `kubectl -n capd-1 get dockermachine`
  - for each of those, delete them by editing it: `kubectl -n capd-1 edit dockermachine xxx` to remove all `finalizer`s
  - verify that all k8s resources got cleaned up: `argocd app list` should produce empty list
  - clean up docker containers: run `docker ps -a` to see all containers. Delete the ones prefixed with `capd-1-` using `docker stop` and `docker rm`
- to clean up the entire testbed, run `testing/teardown_testbed.sh`

