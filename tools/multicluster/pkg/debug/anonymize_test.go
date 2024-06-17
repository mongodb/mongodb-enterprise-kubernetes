package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestNoOpAnonymizer_AnonymizeSecret(t *testing.T) {
	// given
	text := "test"
	anonymizer := NoOpAnonymizer{}

	// when
	result := anonymizer.AnonymizeSecret(&v1.Secret{
		Data: map[string][]byte{
			text: []byte(text),
		},
	})

	// then
	assert.Equal(t, text, string(result.Data[text]))
}

func TestSensitiveDataAnonymizer_AnonymizeSecret(t *testing.T) {
	// given
	text := "test"
	anonymizer := SensitiveDataAnonymizer{}

	// when
	result := anonymizer.AnonymizeSecret(&v1.Secret{
		Data: map[string][]byte{
			text: []byte(text),
		},
	})

	// then
	assert.Equal(t, MASKED_TEXT, string(result.Data[text]))
}
