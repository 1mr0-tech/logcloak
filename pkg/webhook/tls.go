package webhook

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/1mr0-tech/logcloak/pkg/metrics"
)

const tlsSecretName = "logcloak-tls"

// TLSBundle holds PEM-encoded cert material.
type TLSBundle struct {
	CACert  []byte
	TLSCert []byte
	TLSKey  []byte
}

// TLSManager holds the live TLS certificate and rotates it before expiry.
// The running TLS server picks up the new cert without restart via GetCertificate.
type TLSManager struct {
	mu          sync.RWMutex
	cert        *tls.Certificate
	caCert      []byte
	kube        kubernetes.Interface
	namespace   string
	serviceName string
	log         *slog.Logger
}

// NewTLSManager loads or generates TLS credentials and returns a manager
// ready to serve and self-rotate. Call WatchAndRotate in a goroutine after construction.
func NewTLSManager(ctx context.Context, kube kubernetes.Interface, namespace, serviceName string, log *slog.Logger) (*TLSManager, error) {
	secret, err := kube.CoreV1().Secrets(namespace).Get(ctx, tlsSecretName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("get tls secret: %w", err)
	}

	var bundle TLSBundle
	if err == nil && len(secret.Data["tls.crt"]) > 0 {
		bundle = TLSBundle{
			CACert:  secret.Data["ca.crt"],
			TLSCert: secret.Data["tls.crt"],
			TLSKey:  secret.Data["tls.key"],
		}
	} else {
		bundle, err = generateSelfSigned(serviceName, namespace)
		if err != nil {
			return nil, fmt.Errorf("generate cert: %w", err)
		}
		stored, storeErr := storeTLSSecret(ctx, kube, namespace, bundle)
		if storeErr != nil {
			return nil, fmt.Errorf("store tls secret: %w", storeErr)
		}
		// Another replica won the race — use their cert so all replicas share one CA.
		if stored != nil {
			bundle = *stored
		}
	}

	cert, err := loadKeyPairWithLeaf(bundle.TLSCert, bundle.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	mgr := &TLSManager{
		cert:        cert,
		caCert:      bundle.CACert,
		kube:        kube,
		namespace:   namespace,
		serviceName: serviceName,
		log:         log,
	}
	mgr.updateExpiryMetric()
	return mgr, nil
}

// Config returns a *tls.Config whose GetCertificate always returns the live cert,
// allowing zero-downtime rotation without restarting the server.
func (m *TLSManager) Config() *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.getCert,
	}
}

// CACert returns a copy of the current CA certificate PEM for patching the webhook.
func (m *TLSManager) CACert() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]byte, len(m.caCert))
	copy(out, m.caCert)
	return out
}

// WatchAndRotate checks cert expiry every 6 hours and regenerates the cert when
// it is within 30 days of expiry. Stops when ctx is cancelled.
func (m *TLSManager) WatchAndRotate(ctx context.Context, webhookName string) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateExpiryMetric()
			if m.nearExpiry() {
				if err := m.rotate(ctx, webhookName); err != nil {
					m.log.Error("tls rotation failed", "error", err)
				}
			}
		}
	}
}

func (m *TLSManager) getCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cert, nil
}

func (m *TLSManager) nearExpiry() bool {
	m.mu.RLock()
	expiry := m.cert.Leaf.NotAfter
	m.mu.RUnlock()
	return time.Until(expiry) < 30*24*time.Hour
}

func (m *TLSManager) rotate(ctx context.Context, webhookName string) error {
	bundle, err := generateSelfSigned(m.serviceName, m.namespace)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	cert, err := loadKeyPairWithLeaf(bundle.TLSCert, bundle.TLSKey)
	if err != nil {
		return fmt.Errorf("key pair: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tlsSecretName, Namespace: m.namespace},
		Data: map[string][]byte{
			"ca.crt":  bundle.CACert,
			"tls.crt": bundle.TLSCert,
			"tls.key": bundle.TLSKey,
		},
	}
	if _, err = m.kube.CoreV1().Secrets(m.namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update secret: %w", err)
	}

	if err := PatchWebhookCABundle(ctx, m.kube, webhookName, bundle.CACert); err != nil {
		return fmt.Errorf("patch caBundle: %w", err)
	}

	m.mu.Lock()
	m.cert = cert
	m.caCert = bundle.CACert
	m.mu.Unlock()

	m.updateExpiryMetric()
	m.log.Info("tls certificate rotated")
	return nil
}

func (m *TLSManager) updateExpiryMetric() {
	m.mu.RLock()
	expiry := m.cert.Leaf.NotAfter
	m.mu.RUnlock()
	metrics.TLSCertExpiry.Set(float64(expiry.Unix()))
}

func loadKeyPairWithLeaf(certPEM, keyPEM []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func generateSelfSigned(serviceName, namespace string) (TLSBundle, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return TLSBundle{}, err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "logcloak-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return TLSBundle{}, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return TLSBundle{}, err
	}

	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return TLSBundle{}, err
	}
	dnsNames := []string{
		serviceName,
		fmt.Sprintf("%s.%s", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
	}
	srvTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: dnsNames[2]},
		DNSNames:     dnsNames,
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	srvDER, err := x509.CreateCertificate(rand.Reader, srvTemplate, caCert, &srvKey.PublicKey, caKey)
	if err != nil {
		return TLSBundle{}, err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srvDER})
	keyDER, _ := x509.MarshalECPrivateKey(srvKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return TLSBundle{CACert: caPEM, TLSCert: certPEM, TLSKey: keyPEM}, nil
}

// storeTLSSecret creates the TLS secret. If it already exists (another replica won
// the race), it reads and returns the existing bundle so all replicas share one cert.
// Returns (nil, nil) on successful create; (*TLSBundle, nil) when the secret already existed.
func storeTLSSecret(ctx context.Context, kube kubernetes.Interface, namespace string, b TLSBundle) (*TLSBundle, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tlsSecretName, Namespace: namespace},
		Data: map[string][]byte{
			"ca.crt":  b.CACert,
			"tls.crt": b.TLSCert,
			"tls.key": b.TLSKey,
		},
	}
	_, err := kube.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		return nil, nil
	}
	if !errors.IsAlreadyExists(err) {
		return nil, err
	}
	existing, err := kube.CoreV1().Secrets(namespace).Get(ctx, tlsSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("read existing tls secret: %w", err)
	}
	return &TLSBundle{
		CACert:  existing.Data["ca.crt"],
		TLSCert: existing.Data["tls.crt"],
		TLSKey:  existing.Data["tls.key"],
	}, nil
}

// PatchWebhookCABundle updates the caBundle field in a MutatingWebhookConfiguration.
func PatchWebhookCABundle(ctx context.Context, kube kubernetes.Interface, webhookName string, caCert []byte) error {
	caB64 := base64.StdEncoding.EncodeToString(caCert)
	patch := fmt.Sprintf(`[{"op":"replace","path":"/webhooks/0/clientConfig/caBundle","value":%q}]`, caB64)
	_, err := kube.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(
		ctx, webhookName, types.JSONPatchType, []byte(patch), metav1.PatchOptions{},
	)
	return err
}
