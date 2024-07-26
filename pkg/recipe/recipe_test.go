package recipe_test

import (
	"bytes"
	"errors"
	"io"
	"mp/doit/pkg/recipe"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Step1(input io.Reader) (io.Reader, error) {
	calls["recipe_test.Step1"]++
	text, err := io.ReadAll(input)
	if err != nil {
		panic(err)
	}
	if string(text) == ShortCircuit {
		return nil, errors.New(ShortCircuit)
	}
	output := strings.NewReader(strings.ReplaceAll(string(text), GarbageIn, GarbageOut))
	return output, nil
}

func Step2(input io.Reader) (io.Reader, error) {
	calls["recipe_test.Step2"]++
	text, err := io.ReadAll(input)
	if err != nil {
		panic(err)
	}

	if string(text) == TemporaryShortCircuit {
		if ignoreTemporaryShortCircuit {
			text = []byte(GarbageOut)
		} else {
			return nil, errors.New(TemporaryShortCircuit)
		}
	}

	output := bytes.NewReader(text)
	return output, nil
}

var (
	calls                       = make(map[string]int)
	ignoreTemporaryShortCircuit = false

	stepFuncMap = recipe.StepFuncMap{
		"recipe_test.Step1": Step1,
		"recipe_test.Step2": Step2,
	}
)

const (
	GarbageIn             = "garbage in"
	GarbageOut            = "garbage out"
	ShortCircuit          = "short circuit"
	TemporaryShortCircuit = "temporary short circuit"

	recipeYaml = `
name: test recipe
kind: recipe/v1
steps:
- kind: step/v1
  name: recipe_test.Step1
- kind: step/v1
  name: recipe_test.Step2
`
)

func TestNewRecipe(t *testing.T) {
	want := &recipe.Recipe{
		Name: "test recipe",
		Steps: []recipe.Step{
			{
				Kind:     "step/v1",
				Name:     "recipe_test.Step1",
				StepFunc: Step1,
			},
			{
				Kind:     "step/v1",
				Name:     "recipe_test.Step2",
				StepFunc: Step2,
			},
		},
	}

	got, err := recipe.New(strings.NewReader(recipeYaml), stepFuncMap)
	assert.NoError(t, err)
	assert.Equal(t, want.Name, got.Name)
	assert.Len(t, got.Steps, len(want.Steps))
	for i := 0; i < len(want.Steps); i++ {
		assertSameStep(t, want.Steps[i], got.Steps[i])
	}
}

func TestCook(t *testing.T) {
	calls = make(map[string]int)

	r, err := recipe.New(strings.NewReader(recipeYaml), stepFuncMap)
	assert.NoError(t, err)

	out, err := r.Cook(strings.NewReader("spam"))
	assert.NoError(t, err)

	text, err := io.ReadAll(out)
	assert.NoError(t, err)

	assert.Equal(t, "spam", string(text))

	assert.Equal(t, 1, calls["recipe_test.Step1"])
	assert.Equal(t, 1, calls["recipe_test.Step2"])
}

func TestArguments(t *testing.T) {
	r, err := recipe.New(strings.NewReader(recipeYaml), stepFuncMap)
	assert.NoError(t, err)

	out, err := r.Cook(strings.NewReader(GarbageIn))
	assert.NoError(t, err)

	text, err := io.ReadAll(out)
	assert.NoError(t, err)
	assert.Equal(t, GarbageOut, string(text))
}

func TestShortCircuit(t *testing.T) {
	calls = make(map[string]int)

	r, err := recipe.New(strings.NewReader(recipeYaml), stepFuncMap)
	assert.NoError(t, err)

	_, err = r.Cook(strings.NewReader(ShortCircuit))
	assert.Error(t, err)
	assert.Equal(t, 1, calls["recipe_test.Step1"])
	assert.Equal(t, 0, calls["recipe_test.Step2"])
}

func TestRetry(t *testing.T) {
	calls = make(map[string]int)

	r, err := recipe.New(strings.NewReader(recipeYaml), stepFuncMap)
	assert.NoError(t, err)

	_, err = r.Cook(strings.NewReader(TemporaryShortCircuit))
	assert.Error(t, err)
	assert.Equal(t, 1, calls["recipe_test.Step1"])
	assert.Equal(t, 1, calls["recipe_test.Step2"]) // we made it to Step2, but it failed

	calls = make(map[string]int)
	state := r.State()

	ignoreTemporaryShortCircuit = true
	defer func() { ignoreTemporaryShortCircuit = false }()

	r, err = recipe.New(strings.NewReader(recipeYaml), stepFuncMap, recipe.WithState(state))
	assert.NoError(t, err)

	out, err := r.Cook(strings.NewReader(TemporaryShortCircuit))
	assert.NoError(t, err)

	text, err := io.ReadAll(out)
	assert.NoError(t, err)
	assert.Equal(t, GarbageOut, string(text))

	assert.Equal(t, 0, calls["recipe_test.Step1"]) // we didn't re-run Step1
	assert.Equal(t, 1, calls["recipe_test.Step2"]) // this time we made it to Step2
}

func assertSameStep(t *testing.T, want recipe.Step, got recipe.Step) {
	t.Helper()

	wantReader, wantErr := want.Do(strings.NewReader("spam"))
	gotReader, gotErr := got.Do(strings.NewReader("spam"))

	assert.Equal(t, wantErr, gotErr)

	wantBytes, wantErr := io.ReadAll(wantReader)
	assert.NoError(t, wantErr)

	gotBytes, gotErr := io.ReadAll(gotReader)
	assert.NoError(t, gotErr)

	assert.Equal(t, wantBytes, gotBytes)
}
