
set -e
# $1 = expression
# $2 = sleep period
# $3 = iterations
function wait_until()
{
    for i in `seq 1 $3`
    do
        eval $1 && return 0
        echo "Waiting for \"$1\" to evaluate to true ..."
        sleep $2
    done
    echo Timed out waiting for \"$1\"
    return 1
}

repourl=http://172.17.0.1:8188/myrepo.git
repodir=/tmp/arlon-testbed-git-clone/myrepo
repopath=baseclusters/capd-test
baseclusterdir=${repodir}/${repopath}
manifestfile=capd-capi-quickstart-withclusternamelabelremoved.yaml

if ! [ -f ${baseclusterdir}/capd-capi-quickstart-withclusternamelabelremoved.yaml ]; then
    echo adding basecluster directory
    mkdir -p ${baseclusterdir}
    cp testing/${manifestfile} ${baseclusterdir}
    pushd ${baseclusterdir}
    git pull
    git add ${manifestfile}
    git commit -m ${manifestfile}
    git push origin main
    popd
fi

if arlon basecluster validategit --repo-url ${repourl} --repo-path ${repopath} 2> /tmp/stdout.txt; then
    echo ${repopath} already prepped
else
    if ! grep "Error: kustomization.yaml is missing" /tmp/stdout.txt &> /dev/null ; then
        echo unexpected output from validategit, check /tmp/stdout.txt
        exit 1
    fi
    if ! arlon basecluster preparegit --repo-url ${repourl} --repo-path ${repopath}; then
        echo preparegit failed
        exit 1
    fi
fi

arlon cluster create --cluster-name capd-1 --repo-url http://172.17.0.1:8188/myrepo.git --repo-path baseclusters/capd-test --profile dynamic-2-calico

echo '------- waiting for control plane to become ready ---------'
wait_until 'set -o pipefail; clusterctl -n capd-1 describe cluster capd-1-capi-quickstart 2> /dev/null |grep ControlPlane|grep True > /dev/null' 10 30

echo '------- waiting for machinedeployment to become ready (happens when calico is deployed) ---------'
wait_until 'set -o pipefail; clusterctl -n capd-1 describe cluster capd-1-capi-quickstart|grep MachineDeployment|grep True > /dev/null' 10 60

echo '------- waiting for guestbook to become healthy ---------'
wait_until 'set -o pipefail; argocd app list|grep capd-1-guestbook-dynamic|grep Healthy > /dev/null' 10 30

echo '--- test ok ---'
