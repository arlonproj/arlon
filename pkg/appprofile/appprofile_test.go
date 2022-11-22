package appprofile

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	sets "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

type mockArgoClient struct {
	argoclient.Client
}

type mockIoCloser struct {
	io.Closer
}

func (mic *mockIoCloser) Close() error {
	return nil
}

type mockClusterSvcClient struct {
	clusterpkg.ClusterServiceClient
}

func (mac *mockArgoClient) NewClusterClient() (io.Closer, clusterpkg.ClusterServiceClient, error) {
	return &mockIoCloser{}, &mockClusterSvcClient{}, nil
}

type mockApplicationSvcClient struct {
	applicationpkg.ApplicationServiceClient
}

func (mac *mockArgoClient) NewApplicationClient() (io.Closer, applicationpkg.ApplicationServiceClient, error) {
	return &mockIoCloser{}, &mockApplicationSvcClient{}, nil
}

var (
	gClusterList        *v1alpha1.ClusterList
	gApplicationList    *v1alpha1.ApplicationList
	gApplicationSetList *appset.ApplicationSetList
	gProfileList        *arlonv1.AppProfileList
)

func init() {
	gClusterList = &v1alpha1.ClusterList{
		Items: []v1alpha1.Cluster{
			{
				Name:   "cluster-1",
				Server: "cluster-1.local",
			},
			{
				Name:   "cluster-2",
				Server: "cluster-2.local",
			},
		},
	}

	gApplicationList = &v1alpha1.ApplicationList{
		Items: []v1alpha1.Application{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clust-2",
					Labels: map[string]string{
						"arlon-type": "cluster-app",
						"managed-by": "arlon",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clust-1",
					Labels: map[string]string{
						"arlon-type": "cluster-app",
						"managed-by": "arlon",
					},
				},
			},
		},
	}

	gApplicationSetList = &appset.ApplicationSetList{
		Items: []appset.ApplicationSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wordpress",
					Labels: map[string]string{
						"arlon-type": "application",
					},
				},
				Spec: appset.ApplicationSetSpec{
					Generators: []appset.ApplicationSetGenerator{
						{
							List: &appset.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysql",
					Labels: map[string]string{
						"arlon-type": "application",
					},
				},
				Spec: appset.ApplicationSetSpec{
					Generators: []appset.ApplicationSetGenerator{
						{
							List: &appset.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "petclinic",
					Labels: map[string]string{
						"arlon-type": "application",
					},
				},
				Spec: appset.ApplicationSetSpec{
					Generators: []appset.ApplicationSetGenerator{
						{
							List: &appset.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "teamcity",
					Labels: map[string]string{
						"arlon-type": "application",
					},
				},
				Spec: appset.ApplicationSetSpec{
					Generators: []appset.ApplicationSetGenerator{
						{
							List: &appset.ListGenerator{},
						},
					},
				},
			},
		},
	}

	gProfileList = &arlonv1.AppProfileList{
		Items: []arlonv1.AppProfile{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "marketing",
				},
				Spec: arlonv1.AppProfileSpec{
					AppNames: []string{
						"wordpress",
						"nonexistent-1",
						"mysql",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "engineering",
				},
				Spec: arlonv1.AppProfileSpec{
					AppNames: []string{
						"mysql",
						"nonexistent-2",
						"petclinic",
						"nonexistent-1",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "qa",
				},
				Spec: arlonv1.AppProfileSpec{
					AppNames: []string{
						"mysql",
						"teamcity",
					},
				},
			},
		},
	}
}

func (mcsc *mockClusterSvcClient) List(ctx context.Context,
	in *clusterpkg.ClusterQuery,
	opts ...grpc.CallOption) (*v1alpha1.ClusterList, error) {
	return gClusterList, nil
}

func (masc *mockApplicationSvcClient) List(ctx context.Context,
	in *applicationpkg.ApplicationQuery,
	opts ...grpc.CallOption) (*v1alpha1.ApplicationList, error) {
	return gApplicationList, nil
}

type mockCtrlRuntClient struct {
	client.Client
}

func (mcrc *mockCtrlRuntClient) List(ctx context.Context,
	list client.ObjectList, opts ...client.ListOption) error {
	if appSetListPtr, ok := list.(*appset.ApplicationSetList); ok {
		*appSetListPtr = *gApplicationSetList
	} else {
		profileListPtr := list.(*arlonv1.AppProfileList)
		*profileListPtr = *gProfileList
	}
	return nil
}

type mockStatusWriter struct {
	client.StatusWriter
}

func (mcrc *mockCtrlRuntClient) Status() client.StatusWriter {
	return &mockStatusWriter{}
}

func (msw *mockStatusWriter) Update(ctx context.Context, obj client.Object,
	opts ...client.UpdateOption) error {
	if pProfile, ok := obj.(*arlonv1.AppProfile); ok {
		prof := lookupProfile(pProfile.Name)
		if prof == nil {
			return fmt.Errorf("failed to find profile named %s", pProfile.Name)
		}
		*prof = *pProfile
		return nil
	}
	return fmt.Errorf("updating object of that type not supported")
}

func lookupProfile(name string) *arlonv1.AppProfile {
	for i, prof := range gProfileList.Items {
		if prof.Name == name {
			return &gProfileList.Items[i]
		}
	}
	return nil
}

func TestAppProfileReconcileEverything(t *testing.T) {
	logr := zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}))
	var mcr *mockCtrlRuntClient
	var mac *mockArgoClient
	_, err := ReconcileEverything(nil, mcr, mac, logr)
	if err != nil {
		t.Errorf("reconcile error: %s", err)
	}
	assert.Equal(t, gProfileList.Items[0].Status.Health, "degraded")
	assert.True(t, stringSetsEqual(gProfileList.Items[0].Status.InvalidAppNames, []string{"nonexistent-1"}))
	assert.Equal(t, gProfileList.Items[1].Status.Health, "degraded")
	assert.True(t, stringSetsEqual(gProfileList.Items[1].Status.InvalidAppNames, []string{"nonexistent-1", "nonexistent-2"}))
	assert.Equal(t, gProfileList.Items[2].Status.Health, "healthy")
	dumpProfiles(t)
}

func dumpProfiles(t *testing.T) {
	for _, prof := range gProfileList.Items {
		t.Log("profile:", prof)
	}
}

func stringSetsEqual(s1 []string, s2 []string) bool {
	set1 := sets.NewSet[string](s1...)
	set2 := sets.NewSet[string](s2...)
	return set1.Equal(set2)
}
