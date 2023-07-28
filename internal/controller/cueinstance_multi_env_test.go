package controller

import (
	"context"
	"testing"
	"time"

	cueinstancev1a1 "github.com/akirill0v/cue-flux-controller/api/v1alpha1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCueInstanceReconciler_MultiEnv(t *testing.T) {
	g := NewWithT(t)
	id := "builder-" + randStringRunes(5)

	err := createNamespace(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	err = createKubeConfigSecret(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create kubeconfig secret")

	deployNamespace := "cue-multi-env"
	err = createNamespace(deployNamespace)
	g.Expect(err).NotTo(HaveOccurred())

	artifactFile := "instance-" + randStringRunes(5)
	artifactChecksum, err := createArtifact(testServer, "testdata/multi-env", artifactFile)
	g.Expect(err).ToNot(HaveOccurred())

	repositoryName := types.NamespacedName{
		Name:      randStringRunes(5),
		Namespace: id,
	}

	err = applyGitRepository(repositoryName, artifactFile, "main/"+artifactChecksum)
	g.Expect(err).NotTo(HaveOccurred())

	cueInstanceKey := types.NamespacedName{
		Name:      "inst-" + randStringRunes(5),
		Namespace: id,
	}

	cueInstance := &cueinstancev1a1.CueInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cueInstanceKey.Name,
			Namespace: cueInstanceKey.Namespace,
		},
		Spec: cueinstancev1a1.CueInstanceSpec{
			Interval: metav1.Duration{Duration: reconciliationInterval},
			Root:     "./testdata/multi-env",
			Path:     "./infra/dev",
			Exprs: []string{
				"out",
			},
			Tags: []cueinstancev1a1.TagVar{
				{
					Name: "dev",
				},
			},
			KubeConfig: &meta.KubeConfigReference{
				SecretRef: meta.SecretKeyReference{
					Name: "kubeconfig",
				},
			},
			SourceRef: cueinstancev1a1.CrossNamespaceSourceReference{
				Name:      repositoryName.Name,
				Namespace: repositoryName.Namespace,
				Kind:      sourcev1.GitRepositoryKind,
			},
		},
	}

	g.Expect(k8sClient.Create(context.TODO(), cueInstance)).To(Succeed())

	g.Eventually(func() bool {
		var obj cueinstancev1a1.CueInstance
		_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cueInstance), &obj)
		return obj.Status.LastAppliedRevision == "main/"+artifactChecksum
	}, timeout, time.Second).Should(BeTrue())

	ns := &corev1.Namespace{}
	g.Expect(k8sClient.Get(context.TODO(), types.NamespacedName{
		Name: "dev",
	}, ns)).To(Succeed())
}
