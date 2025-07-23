package transform

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCustomTemplate(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "template")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	text := "This is a custom template"
	if _, err := tmpfile.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create a commentTemplate with customTemplate set to the path of the temporary file
	ct := &commentTemplate{
		customTemplate: tmpfile.Name(),
	}

	// Call getCustomTemplate and check if the returned string is the same as the data written to the file
	got, err := ct.getCustomTemplate()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, text, got)

	// Create a commentTemplate with customTemplate set to a non-existing file
	ct = &commentTemplate{
		customTemplate: "cdk diff{{ .TagID }}",
	}

	// Call getCustomTemplate and check if the returned string is the same as the data written to the file
	got, err = ct.getCustomTemplate()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ct.customTemplate, got)

	// Change Tempfile to an unreadable file
	// we expect an error
	err = os.Chmod(tmpfile.Name(), 0000)
	if err != nil {
		t.Fatal(err)
	}
	ct = &commentTemplate{
		customTemplate: tmpfile.Name(),
	}
	_, err = ct.getCustomTemplate()
	assert.Error(t, err)
}

func TestCommentTemplateChooseTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		customTemplate string
		showOverview   bool
		expectedType   reflect.Type
	}{
		{
			name:           "WithCustomTemplate",
			customTemplate: "This is a custom template",
			expectedType:   reflect.TypeOf(CustomTemplate{}),
		},
		{
			name:         "WithDefaultTemplate",
			template:     "default",
			expectedType: reflect.TypeOf(DefaultTemplate{}),
		},
		{
			name:         "WithExtendedTemplate",
			template:     "extended",
			expectedType: reflect.TypeOf(ExtendedTemplate{}),
		},
		{
			name:         "WithNonExistingTemplate",
			template:     "non-existing",
			expectedType: reflect.TypeOf(DefaultTemplate{}),
		},
		{
			name:         "WithShowOverview",
			template:     "default",
			showOverview: true,
			expectedType: reflect.TypeOf(ExtendedTemplate{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := &commentTemplate{
				Template:       tt.template,
				customTemplate: tt.customTemplate,
				ShowOverview:   tt.showOverview,
			}

			got := ct.ChooseTemplate()
			assert.IsType(t, tt.expectedType, reflect.TypeOf(got))
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name        string
		template    commentTemplate
		expected    string
		expectError bool
	}{
		{
			name: "WithValidTemplate",
			template: commentTemplate{
				customTemplate: "cdk diff {{.TagID}}!",
				TagID:          "small",
			},
			expected:    "cdk diff small!",
			expectError: false,
		},
		{
			name: "WithValidTemplateButEmptyTagID",
			template: commentTemplate{
				customTemplate: "cdk diff {{.TagID}}!",
			},
			expected: "cdk diff !",
		},
		{
			name: "WithInvalidTemplate",
			template: commentTemplate{
				customTemplate: "cdk diff {{.TagID}!",
				TagID:          "small",
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "WithCustomSprigTemplate",
			template: commentTemplate{
				customTemplate: "cdk diff {{ upper \"badgers\" }}",
			},
			expected:    "cdk diff BADGERS",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.template.render()
			assert.Equal(t, tt.expected, got)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
