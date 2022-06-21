package webhook

import (
	"context"
	"fmt"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

func (wh *webhook) Rewrite(ctx context.Context, ar v1.AdmissionReview) *v1.AdmissionResponse {
	fmt.Println("check if we should rewrite cluster name")
	// Check if received object
	podResource := metav1.GroupVersionResource{Group: "cluster.x-k8s.io", Version: "v1beta1", Resource: "clusters"}
	if ar.Request.Resource != podResource {
		wh.log.Info("unexpected resource type: %s %s %s",
			ar.Request.Resource.Group,
			ar.Request.Resource.Version,
			ar.Request.Resource.Resource,
		)
		return nil
	}

	fmt.Println("parsing subject object")
	raw := ar.Request.Object.Raw
	clust := capi.Cluster{}
	if _, _, err := wh.decoder.Decode(raw, nil, &clust); err != nil {
		return wh.toV1AdmissionResponse("failed to decode cluster", err)
	}
	reviewResponse := &v1.AdmissionResponse{
		Allowed: true,
	}
	instance := clust.Labels["app.kubernetes.io/instance"]
	if instance == "" {
		wh.log.Info("cluster does not have instance label",
			"clustername", clust.Name)
		return reviewResponse
	}
	var patches []interface{}
	newName := fmt.Sprintf("%s-%s", instance, clust.Name)
	patches = append(patches, map[string]interface{}{
		"op":    "replace",
		"path":  "/metadata/name",
		"value": newName,
	})
	if len(patches) > 0 {
		bs, err := json.Marshal(patches)
		if err != nil {
			return wh.toV1AdmissionResponse("failed to marshal patches", err)
		}
		pt := v1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
		reviewResponse.Patch = bs
	}
	return reviewResponse
}

func (wh *webhook) toV1AdmissionResponse(msg string, err error) *v1.AdmissionResponse {
	wh.log.Error(err, msg)
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
