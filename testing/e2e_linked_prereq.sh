arlon cluster ngupdate ec2-cluster --profile dynamic-2
clusterctl get kubeconfig ec2-cluster-capi-quickstart -n ec2-cluster > arlon-eks.kubeconfig
cp arlon-eks.kubeconfig /home/runner/work/arlon/arlon/kubeconfig 
cp ~/.kube/config ~/.kube/temp.config
cp arlon-eks.kubeconfig ~/.kube/config