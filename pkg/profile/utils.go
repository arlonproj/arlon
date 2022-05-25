package profile

import (
	"context"
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/controller"
	sets "github.com/deckarep/golang-set"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type AugmentedProfile struct {
	arlonv1.Profile
	Legacy bool // whether the profile is stored as a configmap
}

// -----------------------------------------------------------------------------

func List(config *restclient.Config, ns string) (plist []AugmentedProfile, err error) {
	cli, err := controller.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var profList arlonv1.ProfileList
	err = cli.List(context.Background(), &profList, &client.ListOptions{
		Namespace: ns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %s", err)
	}
	for _, prof := range profList.Items {
		plist = append(plist, AugmentedProfile{Profile: prof})
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(ns)
	opts := metav1.ListOptions{
		LabelSelector: "managed-by=arlon,arlon-type=profile",
	}
	configMaps, err := configMapsApi.List(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list configMaps: %s", err)
	}
	for _, cm := range configMaps.Items {
		prof, err := FromConfigMap(&cm)
		if err != nil {
			return nil, fmt.Errorf("failed to convert configmap to profile: %s", err)
		}
		plist = append(plist, AugmentedProfile{
			Profile: *prof,
			Legacy:  true,
		})
	}
	return
}

// -----------------------------------------------------------------------------

// FromConfigMap retrieves a Gen1 profile, which is stored stored in a config map
func FromConfigMap(cm *v1.ConfigMap) (*arlonv1.Profile, error) {
	if cm.Labels["arlon-type"] != "profile" {
		return nil, fmt.Errorf("config map %s is missing arlon-type label", cm.Name)
	}
	bundleList := StringListFromCommaSeparated(cm.Data["bundles"])
	tagList := StringListFromCommaSeparated(cm.Data["tags"])
	return &arlonv1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: cm.Name,
		},
		Spec: arlonv1.ProfileSpec{
			Description:  cm.Data["description"],
			Tags:         tagList,
			Bundles:      bundleList,
			RepoUrl:      cm.Data["repo-url"],
			RepoPath:     cm.Data["repo-path"],
			RepoRevision: cm.Data["repo-branch"],
		},
	}, nil
}

// -----------------------------------------------------------------------------

func Get(config *restclient.Config, name string, ns string) (*arlonv1.Profile, error) {
	cli, err := controller.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	objKey := client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	var prof arlonv1.Profile
	err = cli.Get(context.Background(), objKey, &prof)
	if err == nil {
		return &prof, nil
	}
	if !apierr.IsNotFound(err) {
		return nil, fmt.Errorf("unexpected error looking up profile: %s", err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	cm, err := getProfileConfigMap(name, corev1, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile from configmap: %s", err)
	}
	return FromConfigMap(cm)
}

// -----------------------------------------------------------------------------

func getProfileConfigMap(
	profileName string,
	corev1 corev1types.CoreV1Interface,
	arlonNs string,
) (prof *v1.ConfigMap, err error) {
	if profileName == "" {
		return nil, fmt.Errorf("profile name not specified")
	}
	configMapsApi := corev1.ConfigMaps(arlonNs)
	profileConfigMap, err := configMapsApi.Get(context.Background(), profileName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get profile configmap: %s", err)
	}
	if profileConfigMap.Labels["arlon-type"] != "profile" {
		return nil, fmt.Errorf("profile configmap does not have expected label")
	}
	return profileConfigMap, nil
}

// -----------------------------------------------------------------------------

func StringListFromCommaSeparated(cs string) (sl []string) {
	if cs == "" {
		return
	}
	for _, item := range strings.Split(cs, ",") {
		sl = append(sl, item)
	}
	return
}

func CommaSeparatedFromStringList(sl []string) string {
	var s string
	for i, item := range sl {
		if i > 0 {
			item = "," + item
		}
		s = s + item
	}
	return s
}

// -----------------------------------------------------------------------------

// stringListsEquivalent returns true if the set of items defined by
// the string list s1 is equal to the set defined by s2
func stringListsEquivalent(sl1 []string, sl2 []string) bool {
	s1 := sets.NewSetFromSlice(stringSliceToInterfaceSlice(sl1))
	s2 := sets.NewSetFromSlice(stringSliceToInterfaceSlice(sl2))
	return s1.Equal(s2)
}

// See https://go.dev/doc/faq#convert_slice_of_interface
func stringSliceToInterfaceSlice(t []string) []interface{} {
	s := make([]interface{}, len(t))
	for i, v := range t {
		s[i] = v
	}
	return s
}

// -----------------------------------------------------------------------------

func GetAugmented(config *restclient.Config, name string, ns string) (*AugmentedProfile, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(ns)
	cm, err := configMapsApi.Get(context.Background(), name, metav1.GetOptions{})
	if err == nil {
		prof, err := FromConfigMap(cm)
		if err != nil {
			return nil, fmt.Errorf("failed to convert configmap to profile: %s", err)
		}
		return &AugmentedProfile{
			Profile: *prof,
			Legacy:  true,
		}, nil
	}
	prof, err := Get(config, name, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %s", err)
	}
	return &AugmentedProfile{Profile: *prof}, nil
}
