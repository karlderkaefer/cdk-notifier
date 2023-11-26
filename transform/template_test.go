package transform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommentTemplateChooseTemplate(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		showOverview bool
		want         TemplateStrategy
	}{
		{
			name:     "default template",
			template: "default",

			want: DefaultTemplate{},
		},
		{
			name:     "extended template",
			template: "extended",
			want:     ExtendedTemplate{},
		},
		{
			name:     "non-existing template",
			template: "non-existing",
			want:     DefaultTemplate{},
		},
		{
			name:         "extended template with activated overview",
			template:     "default",
			showOverview: true,
			want:         ExtendedTemplate{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := &commentTemplate{
				Template:     tt.template,
				ShowOverview: tt.showOverview,
			}
			got := ct.ChooseTemplate()
			assert.IsType(t, tt.want, got)
		})
	}
}

func TestGetCustomTemplate(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "template")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

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

}
