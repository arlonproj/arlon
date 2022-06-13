package webhook

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/log"
	"github.com/go-logr/logr"
	"io/ioutil"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/wh.log/v2"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type webhook struct {
	localClient client.Client
	scheme      *runtime.Scheme
	decoder     runtime.Decoder
	log         logr.Logger
}

func newWebhook(localClient client.Client, runtimeScheme *runtime.Scheme) *webhook {
	return &webhook{
		localClient: localClient,
		scheme:      runtimeScheme,
		decoder:     serializer.NewCodecFactory(runtimeScheme).UniversalDeserializer(),
		log:         log.GetLogger(),
	}
}

func (wh *webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err, code := wh.serveHTTP(w, r)
	if err != nil {
		msg := fmt.Sprintf("error: %s", err)
		wh.log.Info(msg)
	}
}

func (wh *webhook) serveHTTP(w http.ResponseWriter, r *http.Request) (error, int) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return fmt.Errorf("unexpected content type: %s", contentType),
			http.StatusBadRequest
	}

	wh.log.Info(fmt.Sprintf("handling request: %s", body))

	obj, gvk, err := wh.decoder.Decode(body, nil, nil)
	if err != nil {

		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		wh.log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1.AdmissionReview)
		if !ok {
			wh.log.Errorf("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = wh.RedirectPod(r.Context(), *requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		wh.log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	wh.log.V(2).Info(fmt.Sprintf("sending response: %v", responseObj))
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		wh.log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		wh.log.Error(err)
	}
}
