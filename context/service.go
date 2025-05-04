package context

import "github.com/tmc/langchaingo/llms"

type serviceManagerContext struct {
	primaryModel llms.Model
}

func NewServiceManagerContext(primaryModel llms.Model) *serviceManagerContext {
	return &serviceManagerContext{
		primaryModel,
	}
}

func (c *serviceManagerContext) PrimaryModel() llms.Model {
	return c.primaryModel
}
