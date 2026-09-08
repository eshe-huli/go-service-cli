package app

import (
	"context"
	"example.com/greeter/internal/greetings/internal/domain"
)

type GreetPersonDeps struct{}
type GreetPerson struct{ deps GreetPersonDeps }

func NewGreetPerson(deps GreetPersonDeps) *GreetPerson { return &GreetPerson{deps: deps} }

func (h *GreetPerson) Execute(ctx context.Context, input GreetPersonInput) (GreetPersonOutput, error) {
	if err := ctx.Err(); err != nil {
		return GreetPersonOutput{}, err
	}
	if err := input.Validate(); err != nil {
		return GreetPersonOutput{}, err
	}
	return GreetPersonOutput{Message: domain.Greeting(input.Name)}, nil
}
