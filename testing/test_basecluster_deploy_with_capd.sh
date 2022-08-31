
set -e

repodir=/tmp/arlon-testbed-git-clone/myrepo
baseclusterdir=${repodir}/capd-test
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

