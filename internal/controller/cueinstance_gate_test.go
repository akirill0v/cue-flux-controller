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

func TestCueInstanceReconciler_Gates(t *testing.T) {
	g := NewWithT(t)
	id := "builder-" + randStringRunes(5)

	err := createNamespace(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	err = createKubeConfigSecret(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create kubeconfig secret")

	deployNamespace := "cue-gate"
	err = createNamespace(deployNamespace)
	g.Expect(err).NotTo(HaveOccurred())

	artifactFile := "instance-" + randStringRunes(5)
	artifactChecksum, err := createArtifact(testServer, "testdata/gates", artifactFile)
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

	tagName := "podinfo" + randStringRunes(5)

	cueInstance := &cueinstancev1a1.CueInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cueInstanceKey.Name,
			Namespace: cueInstanceKey.Namespace,
		},
		Spec: cueinstancev1a1.CueInstanceSpec{
			Interval: metav1.Duration{Duration: reconciliationInterval},
			Root:     "./testdata/gates",
			Exprs: []string{
				"out",
			},
			Gates: []cueinstancev1a1.GateExpr{
				{
					Name: "deploy",
					Expr: "deployGate",
				},
			},
			Tags: []cueinstancev1a1.TagVar{
				{
					Name:  "gate",
					Value: "tummy",
				},
				{
					Name:  "name",
					Value: tagName,
				},
				{
					Name:  "namespace",
					Value: deployNamespace,
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

	cm := &corev1.ConfigMap{}
	g.Expect(k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      tagName,
		Namespace: deployNamespace,
	}, cm)).ToNot(Succeed())

	cinst := &cueinstancev1a1.CueInstance{}
	g.Expect(k8sClient.Get(context.TODO(), cueInstanceKey, cinst)).To(Succeed())

	patch := client.MergeFrom(cinst.DeepCopy())

	cinst.Spec.Tags[0] = cueinstancev1a1.TagVar{
		Name:  "gate",
		Value: "dummy",
	}

	g.Expect(k8sClient.Patch(context.TODO(), cinst, patch)).To(Succeed())

	g.Eventually(func() bool {
		key := types.NamespacedName{
			Name:      tagName,
			Namespace: deployNamespace,
		}
		if err := k8sClient.Get(context.TODO(), key, cm); err != nil {
			return false
		}
		return true
	}, timeout, time.Second).Should(BeTrue())
}
