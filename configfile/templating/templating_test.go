package templating

import (
	"os"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {
	os.Setenv("ALWAYS_THERE", "always_there")

	source := []byte(`
K={{ env "ALWAYS_THERE" }}
K={{ env "NONEXISTING" }}
K={{ .NONEXISTING }}
K={{ default .NonExisting "default value" }}
K={{ default (env "ALWAYS_THERE") }}
K={{ required (default ( .NotValid ) "Valid") }}
K={{ default (env "NONEXISTING") "default value" }}
`)

	correctOutput := []byte(`
K=always_there
K=
K=<no value>
K=default value
K=always_there
K=Valid
K=default value
`)

	result, err := GenerateTemplate(source)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(correctOutput) {
		t.Fatalf("Result:\n%s\n==== is not equal to correct template output:\n%s\n", result, correctOutput)
	}
}

func TestFailedRequired(t *testing.T) {
	source := []byte(`{{ required ( .NotValid ) }}`)
	_, err := GenerateTemplate(source)

	if err == nil {
		t.Fail()
		t.Log("Required failed to see pickup a failed value")
	}

}
