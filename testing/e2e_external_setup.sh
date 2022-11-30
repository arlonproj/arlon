#!/usr/bin/env bash

#Create profile and bundle
arlon bundle create xenial --tags test,xenial --desc "Xenial pod" --repo-path bundles/xenial
arlon profile create dynamic-1 --repo-base-path profiles --bundles xenial --desc "dynamic test 1" --tags examples

externalCluster='external1'
# External kind cluster 
kind create cluster --name $externalCluster
mkdir ~/external
kind get kubeconfig --internal --name $externalCluster > ~/external/external-kc

kubectl config use-context kind-$externalCluster
endpoint=$(kubectl get endpoints -o jsonpath="{.items[0].subsets[0].addresses[0].ip}")

sed -i "s|$externalCluster-control-plane:6443|${endpoint}:6443|" ~/external/external-kc

kubectl config use-context cluster
argocd cluster add kind-$externalCluster --kubeconfig ~/external/external-kc -y

if argocd cluster list | grep kind-$externalCluster > /dev/null ; then
    echo "External cluster added to argocd"
else
    echo "External cluster not added to argocd....Exiting"
    exit 1
fi
