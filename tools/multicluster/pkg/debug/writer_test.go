package debug

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWriteToFile(t *testing.T) {
	//setup
	uniqueTempDir, err := os.MkdirTemp(os.TempDir(), "*-TestWriteToFile")
	assert.NoError(t, err)
	defer os.RemoveAll(uniqueTempDir)

	//given
	testNamespace := "testNamespace"
	testContext := "testContext"
	testError := fmt.Errorf("test")
	testSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"test": []byte("test"),
		},
	}
	testFile := RawFile{
		Name:    "testFile",
		content: []byte("test"),
	}
	collectionResult := CollectionResult{
		kubeResources: []runtime.Object{testSecret},
		rawObjects:    []RawFile{testFile},
		errors:        []error{testError},
		namespace:     testNamespace,
		context:       testContext,
	}
	outputFiles := []string{"testContext-testNamespace-txt-testFile.txt", "testContext-testNamespace-v1.Secret-test-secret.yaml"}

	//when
	path, compressedFile, err := WriteToFile(uniqueTempDir, collectionResult)
	defer os.RemoveAll(path) // This is fine as in case of an empty path, this does nothing
	defer os.RemoveAll(compressedFile)

	//then
	assert.NoError(t, err)
	assert.NotNil(t, path)
	assert.NotNil(t, compressedFile)

	files, err := os.ReadDir(uniqueTempDir)
	assert.NoError(t, err)
	assert.Equal(t, len(outputFiles), len(files))
	for _, outputFile := range outputFiles {
		found := false
		for _, file := range files {
			if strings.Contains(file.Name(), outputFile) {
				found = true
				break
			}
		}
		assert.Truef(t, found, "File %s not found", outputFile)
	}
	_, err = os.Stat(compressedFile)
	assert.NoError(t, err)
}
