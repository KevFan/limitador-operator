package limitador

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/log"
)

func TestRedisCachedDeploymentOptions(t *testing.T) {
	var (
		namespace = "some-ns"
	)

	logger := log.Log.WithName("redis_deployment_test")
	baseCtx := context.Background()
	ctx := logr.NewContext(baseCtx, logger)

	clientFactory := func(subT *testing.T, objs []client.Object) client.Client {
		s := scheme.Scheme
		err := appsv1.AddToScheme(s)
		assert.NilError(subT, err)

		// Create a fake client to mock API calls.
		clBuilder := fake.NewClientBuilder()
		return clBuilder.WithObjects(objs...).Build()
	}

	t.Run("redis secretRef missing", func(subT *testing.T) {
		cl := clientFactory(subT, nil)
		emptyRedisObj := limitadorv1alpha1.RedisCached{}
		_, err := RedisCachedDeploymentOptions(ctx, cl, namespace, emptyRedisObj)
		assert.Error(subT, err, "there's no ConfigSecretRef set")
	})

	t.Run("redis secret resource missing", func(subT *testing.T) {
		cl := clientFactory(subT, nil)
		redisObj := limitadorv1alpha1.RedisCached{
			ConfigSecretRef: &v1.ObjectReference{Name: "notexisting", Namespace: namespace},
		}
		_, err := RedisCachedDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.Assert(subT, errors.IsNotFound(err))
	})

	t.Run("redis secret does not have URL field", func(subT *testing.T) {
		emptySecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redisSecret", Namespace: namespace},
			StringData: map[string]string{},
			Data:       map[string][]byte{},
			Type:       v1.SecretTypeOpaque,
		}
		cl := clientFactory(subT, []client.Object{emptySecret})
		redisObj := limitadorv1alpha1.RedisCached{
			ConfigSecretRef: &v1.ObjectReference{Name: "redisSecret", Namespace: namespace},
		}
		_, err := RedisCachedDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.Error(subT, err, "the storage config Secret doesn't have the `URL` field")
	})

	t.Run("basic redis options", func(subT *testing.T) {
		redisSecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redisSecret", Namespace: namespace},
			StringData: map[string]string{"URL": "redis://example.com:6379"},
			Type:       v1.SecretTypeOpaque,
		}
		redisSecret.Data = helperGetSecretDataFromStringData(redisSecret.StringData)

		cl := clientFactory(subT, []client.Object{redisSecret})
		redisObj := limitadorv1alpha1.RedisCached{
			ConfigSecretRef: &v1.ObjectReference{Name: "redisSecret", Namespace: namespace},
		}
		options, err := RedisCachedDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Command: []string{"redis_cached", "$(LIMITADOR_OPERATOR_REDIS_URL)"},
			},
		)
	})

	t.Run("redis cache options", func(subT *testing.T) {
		redisSecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redisSecret", Namespace: namespace},
			StringData: map[string]string{"URL": "redis://example.com:6379"},
			Type:       v1.SecretTypeOpaque,
		}
		redisSecret.Data = helperGetSecretDataFromStringData(redisSecret.StringData)

		cl := clientFactory(subT, []client.Object{redisSecret})
		redisObj := limitadorv1alpha1.RedisCached{
			ConfigSecretRef: &v1.ObjectReference{Name: "redisSecret", Namespace: namespace},
			Options: &limitadorv1alpha1.RedisCachedOptions{
				TTL:         &[]int{1}[0],
				Ratio:       &[]int{2}[0],
				FlushPeriod: &[]int{3}[0],
				MaxCached:   &[]int{4}[0],
			},
		}
		options, err := RedisCachedDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Command: []string{
					"redis_cached",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
					"--ttl", "1",
					"--ratio", "2",
					"--flush-period", "3",
					"--max-cached", "4",
				},
			},
		)
	})
}
