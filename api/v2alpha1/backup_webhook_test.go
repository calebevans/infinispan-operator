package v2alpha1

import (
	"errors"
	"time"

	v1 "github.com/infinispan/infinispan-operator/api/v1"
	"github.com/infinispan/infinispan-operator/controllers/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// +kubebuilder:scaffold:imports
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("Backup Webhook", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1

	key := types.NamespacedName{
		Name:      "backup-envtest",
		Namespace: "default",
	}

	AfterEach(func() {
		// Delete created Backup resources
		By("Expecting to delete successfully")
		Eventually(func() error {
			f := &Backup{}
			if err := k8sClient.Get(ctx, key, f); err != nil {
				var statusError *k8serrors.StatusError
				if !errors.As(err, &statusError) {
					return err
				}
				// If the Backup does not exist, do nothing
				if statusError.ErrStatus.Code == 404 {
					return nil
				}
			}
			return k8sClient.Delete(ctx, f)
		}, timeout, interval).Should(Succeed())

		By("Expecting to delete finish")
		Eventually(func() error {
			f := &Backup{}
			return k8sClient.Get(ctx, key, f)
		}, timeout, interval).ShouldNot(Succeed())
	})

	Context("Backup", func() {
		It("Should create successfully", func() {

			created := &Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: BackupSpec{
					Cluster: "some-cluster",
				},
			}

			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			updated := &Backup{}
			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			Expect(updated.Spec.Container.Memory).Should(Equal(constants.DefaultMemorySize.String()))
		})

		It("Should return error if required fields not provided", func() {

			rejected := &Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: BackupSpec{},
			}

			err := k8sClient.Create(ctx, rejected)
			expectInvalidErrStatus(err, statusDetailCause{metav1.CauseTypeFieldValueRequired, "spec.cluster", "'spec.cluster' must be configured"})
		})

		It("Should return error if any spec value is updated", func() {

			created := &Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: BackupSpec{
					Cluster: "some-cluster",
				},
			}

			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			// Ensure Spec is immutable on update
			updated := &Backup{}

			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Spec.Cluster = "New Cluster"

			cause := statusDetailCause{"FieldValueForbidden", "spec", "The Backup spec is immutable and cannot be updated after initial Backup creation"}
			expectInvalidErrStatus(k8sClient.Update(ctx, updated), cause)

			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Spec.Container = v1.InfinispanContainerSpec{CPU: "1"}
			expectInvalidErrStatus(k8sClient.Update(ctx, updated), cause)

			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Spec.Resources = &BackupResources{}
			expectInvalidErrStatus(k8sClient.Update(ctx, updated), cause)

			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Spec.Volume = BackupVolumeSpec{}
			expectInvalidErrStatus(k8sClient.Update(ctx, updated), cause)
		})

		It("Should transform deprecated fields", func() {

			created := &Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: BackupSpec{
					Cluster: "some-cluster",
					Resources: &BackupResources{
						CacheConfigs: []string{"cache-name"},
						Scripts:      []string{"script-name"},
					},
				},
			}

			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			updated := &Backup{}
			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			Expect(updated.Spec.Container.Memory).Should(Equal(constants.DefaultMemorySize.String()))
			Expect(updated.Spec.Resources.CacheConfigs).Should(BeNil())
			Expect(updated.Spec.Resources.Templates).Should(HaveLen(1))
			Expect(updated.Spec.Resources.Scripts).Should(BeNil())
			Expect(updated.Spec.Resources.Tasks).Should(HaveLen(1))
		})
	})
})
