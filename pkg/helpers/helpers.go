package helpers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DeleteTagAnnotation = "limitador.kuadrant.io/delete"
)

func ObjectInfo(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
}

func TagObjectToDelete(obj client.Object) {
	// Add custom annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		obj.SetAnnotations(annotations)
	}
	annotations[DeleteTagAnnotation] = "true"
}

func IsObjectTaggedToDelete(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	annotation, ok := annotations[DeleteTagAnnotation]
	return ok && annotation == "true"
}
