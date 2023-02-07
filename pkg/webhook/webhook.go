package webhook

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/log"
	"github.com/go-logr/logr"
	"io"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
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
		http.Error(w, msg, code)
		return
	}
}

func (wh *webhook) serveHTTP(w http.ResponseWriter, r *http.Request) (error, int) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
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
		return fmt.Errorf("request could not be decoded: %s", err),
			http.StatusBadRequest
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1.AdmissionReview)
		if !ok {
			return fmt.Errorf("expected v1.AdmissionReview but got: %T", obj),
				http.StatusBadRequest
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = wh.Rewrite(r.Context(), *requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		return fmt.Errorf("unsupported group version kind: %v", gvk),
			http.StatusBadRequest
	}

	wh.log.V(2).Info(fmt.Sprintf("sending response: %v", responseObj))
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %s", err),
			http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		return fmt.Errorf("failed to write response: %s", err),
			http.StatusInternalServerError
	}
	return nil, 0
}
