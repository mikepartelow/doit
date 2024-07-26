package recipe

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type StepFunc func(io.Reader) (io.Reader, error)
type StepFuncMap map[string]StepFunc

type Step struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
	// TODO: .Retries .Timeout
	StepFunc StepFunc
}

func (s *Step) Do(input io.Reader) (io.Reader, error) {
	return s.StepFunc(input)
}

type Action struct {
	StepIndex int    `yaml:"step_index"`
	StepName  string `yaml:"step_name"`
	Input     []byte `yaml:"input"`
	Output    []byte `yaml:"output"`
	Err       string `yaml:"error"`
}

type State struct {
	Actions []Action `yaml:"actions"`
}

type Option func(*Recipe)

// WithState initializes a recipe with the given existing state
func WithState(s *State) func(*Recipe) {
	return func(r *Recipe) {
		r.prevState = s
	}
}

type Recipe struct {
	Kind  string `yaml:"kind"`
	Name  string `yaml:"name"`
	Steps []Step `yaml:"steps"`

	prevState *State
	state     *State
}

// New returns a new Recipe.
// stepFuncMap maps step.name to functions of type StepFunc in the local binary.
func New(recipeYaml io.Reader, stepFuncMap StepFuncMap, options ...Option) (*Recipe, error) {
	var r Recipe
	for _, opt := range options {
		opt(&r)
	}
	if r.state == nil {
		r.state = &State{}
	}

	// TODO: if len(state.actions) != 0, check that the actions actually match the pipeline in recipeYaml!

	err := yaml.NewDecoder(recipeYaml).Decode(&r)
	if err == nil {
		if r.Kind != "recipe/v1" {
			return nil, fmt.Errorf("unsupported recipe kind: %q", r.Kind)
		}
		for i := range r.Steps {
			s := &r.Steps[i]
			if s.Kind != "step/v1" {
				return nil, fmt.Errorf("unsupported step kind: %q", s.Kind)
			}
			s.StepFunc = stepFuncMap[s.Name]
		}
	}

	return &r, err
}

// Cook executes the steps in a Recipe, passing input to the first step, and passing the output of each step to the subsequent step.
// Cook records pipeline state that can be retrieved with Recipe.State(), and passed to New() to resume a partially executed pipeline.
// If the Recipe was created with existing state, steps in the state that already passed will not be executed. The output of the last
// passing step in the state will be passed as input to the next unexecuted step in the Recipe.
func (r *Recipe) Cook(input io.Reader) (io.Reader, error) {
	var output io.Reader

	output = input

	stepIdx := 0
	if r.prevState != nil {
		outputIdx := -1
		for actionIdx, a := range r.prevState.Actions {
			if a.Err != "" {
				stepIdx = a.StepIndex
				break
			} else {
				outputIdx = actionIdx
			}
		}
		if outputIdx >= 0 {
			output = bytes.NewReader(r.prevState.Actions[outputIdx].Output)
		}
	}

	for ; stepIdx < len(r.Steps); stepIdx++ {
		step := r.Steps[stepIdx]

		input, inputText := mustDupReader(output)

		var err error
		output, err = step.Do(input)

		var outputText []byte
		if err == nil {
			output, outputText = mustDupReader(output)
		}

		action := Action{
			StepIndex: stepIdx,
			StepName:  step.Name,
			Input:     inputText,
			Output:    outputText,
		}
		if err != nil {
			action.Err = err.Error()
		}
		r.state.Actions = append(r.state.Actions, action)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

// State returns the Recipe's state.
// The state can later be passed to New() to resume a partially executed pipeline.
func (r *Recipe) State() *State {
	return r.state
}

func mustDupReader(r io.Reader) (io.Reader, []byte) {
	text, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(text), text
}
