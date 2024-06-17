package debug

import v1 "k8s.io/api/core/v1"

const (
	MASKED_TEXT = "***MASKED***"
)

type Anonymizer interface {
	AnonymizeSecret(secret *v1.Secret) *v1.Secret
}

var _ Anonymizer = &NoOpAnonymizer{}

type NoOpAnonymizer struct{}

func (n *NoOpAnonymizer) AnonymizeSecret(secret *v1.Secret) *v1.Secret {
	return secret
}

var _ Anonymizer = &SensitiveDataAnonymizer{}

type SensitiveDataAnonymizer struct{}

func (n *SensitiveDataAnonymizer) AnonymizeSecret(secret *v1.Secret) *v1.Secret {
	for key := range secret.Data {
		secret.Data[key] = []byte(MASKED_TEXT)
	}
	return secret
}
