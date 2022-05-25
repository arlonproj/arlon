package profile

import (
	"context"
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/controller"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// Updates a profile to the specified set of bundles. Tags and description
// may also be updated.
// If bundlesPtr is nil, no change is made to the bundle set. Otherwise,
// *bundlesPtr specifies the new set.
func Update(
	config *restclient.Config,
	argocdNs string,
	arlonNs string,
	profileName string,
	bundlesPtr *string,
	desc string,
	tags string,
	overrides []arlonv1.Override,
) (dirty bool, err error) {
	prof, err := GetAugmented(config, profileName, arlonNs)
	if err != nil {
		return false, fmt.Errorf("failed to get augmented profile: %s", err)
	}
	if prof.Legacy {
		return false, fmt.Errorf("cannot update a legacy (gen1) profile")
	}
	if desc != "" && desc != prof.Spec.Description {
		prof.Spec.Description = desc
		dirty = true
	}
	if tags != "" && !stringListsEquivalent(StringListFromCommaSeparated(tags),
		prof.Spec.Tags) {
		prof.Spec.Tags = StringListFromCommaSeparated(tags)
		dirty = true
	}
	var bundles string
	if bundlesPtr == nil {
		bundles = CommaSeparatedFromStringList(prof.Spec.Bundles)
	} else {
		bundles = *bundlesPtr
	}
	if !stringListsEquivalent(StringListFromCommaSeparated(bundles), prof.Spec.Bundles) {
		prof.Spec.Bundles = StringListFromCommaSeparated(bundles)
		dirty = true
	}
	// A new override replaces an existing one if bundle and key match,
	// otherwise it is added to the list.
	for _, o := range overrides {
		found := false
		for i, x := range prof.Spec.Overrides {
			if o.Bundle == x.Bundle && o.Key == x.Key {
				found = true
				if o.Value != x.Value {
					// Replace existing value
					//x.Value = o.Value
					prof.Spec.Overrides[i].Value = o.Value
					dirty = true
				} // else, nothing has changed
				break
			}
		}
		if !found {
			// Didn't find a matching one, so append it to the list
			prof.Spec.Overrides = append(prof.Spec.Overrides, o)
			dirty = true
		}
	}
	if !dirty {
		return
	}
	if prof.Spec.RepoUrl != "" {
		// Dynamic profile needs updating in git
		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			return false, fmt.Errorf("failed to get kube client: %s", err)
		}
		err = createInGit(kubeClient, &prof.Profile, argocdNs, arlonNs)
		if err != nil {
			return false, fmt.Errorf("failed to update dynamic profile in git: %s", err)
		}
	}
	cli, err := controller.NewClient(config)
	if err != nil {
		return false, fmt.Errorf("failed to get new controller runtime client: %s", err)
	}
	err = cli.Update(context.Background(), &prof.Profile)
	if err != nil {
		return false, fmt.Errorf("failed to update profile: %s", err)
	}
	return
}
