// Copyright 2020 Orange SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package apis

package certmanagerpki

import (
	"context"
	"reflect"
	"testing"

	"github.com/Orange-OpenSource/nifikop/api/v1alpha1"
	"github.com/Orange-OpenSource/nifikop/pkg/errorfactory"
	certutil "github.com/Orange-OpenSource/nifikop/pkg/util/cert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func newMockUser() *v1alpha1.NifiUser {
	user := &v1alpha1.NifiUser{}
	user.Name = "test-user"
	user.Namespace = "test-namespace"
	user.Spec = v1alpha1.NifiUserSpec{SecretName: "test-secret", IncludeJKS: true}
	return user
}

func newMockUserSecret() *corev1.Secret {
	secret := &corev1.Secret{}
	secret.Name = "test-secret"
	secret.Namespace = "test-namespace"
	cert, key, _, _ := certutil.GenerateTestCert()
	secret.Data = map[string][]byte{
		corev1.TLSCertKey:         cert,
		corev1.TLSPrivateKeyKey:   key,
		v1alpha1.TLSJKSKeyStore:   []byte("testkeystore"),
		v1alpha1.PasswordKey:      []byte("testpassword"),
		v1alpha1.TLSJKSTrustStore: []byte("testtruststore"),
		v1alpha1.CoreCACertKey:    cert,
	}
	return secret
}

func TestFinalizeUserCertificate(t *testing.T) {
	manager := newMock(newMockCluster())
	if err := manager.FinalizeUserCertificate(context.Background(), &v1alpha1.NifiUser{}); err != nil {
		t.Error("Expected no error, got:", err)
	}
}

func TestReconcileUserCertificate(t *testing.T) {
	manager := newMock(newMockCluster())
	ctx := context.Background()

	manager.client.Create(context.TODO(), newMockUser())
	if _, err := manager.ReconcileUserCertificate(ctx, newMockUser(), scheme.Scheme); err == nil {
		t.Error("Expected resource not ready error, got nil")
	} else if reflect.TypeOf(err) != reflect.TypeOf(errorfactory.ResourceNotReady{}) {
		t.Error("Expected resource not ready error, got:", reflect.TypeOf(err))
	}
	if err := manager.client.Delete(context.TODO(), newMockUserSecret()); err != nil {
		t.Error("could not delete test secret")
	}
	if err := manager.client.Create(context.TODO(), newMockUserSecret()); err != nil {
		t.Error("could not update test secret")
	}
	certificate, err := manager.ReconcileUserCertificate(ctx, newMockUser(), scheme.Scheme)
	if err != nil {
		t.Error("Expected no error, got:", err)
	}

	// Ensure data are goods
	_, err = certutil.DecodeCertificate(certificate.CA)
	if err != nil {
		t.Error("Expected no error while Decoding CA, got:", err)
	}
	_, err = certutil.DecodeCertificate(certificate.Certificate)
	if err != nil {
		t.Error("Expected no error while Decoding Cert, got:", err)
	}
	_, err = certutil.DecodeKey(certificate.Key)
	if err != nil {
		t.Error("Expected no error while Decoding Key, got:", err)
	}
	// Test error conditions
	manager = newMock(newMockCluster())
	manager.client.Create(context.TODO(), newMockUser())
	manager.client.Create(context.TODO(), manager.clusterCertificateForUser(newMockUser(), scheme.Scheme))
	if _, err := manager.ReconcileUserCertificate(ctx, newMockUser(), scheme.Scheme); err == nil {
		t.Error("Expected  error, got nil")
	}
}
