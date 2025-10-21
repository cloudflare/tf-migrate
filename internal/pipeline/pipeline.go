package pipeline

import (
	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

type Pipeline struct {
	handler  interfaces.TransformationHandler
	registry *registry.StrategyRegistry
}

func BuildConfigPipeline(reg *registry.StrategyRegistry) *Pipeline {
	return NewPipelineBuilder(reg).
		With(Preprocess).
		With(Parse).
		With(TransformResources).
		With(Format).
		Build()
}

func BuildStatePipeline(reg *registry.StrategyRegistry) *Pipeline {
	return NewPipelineBuilder(reg).
		With(TransformState).
		With(FormatState).
		Build()
}

func (p *Pipeline) Transform(content []byte, filename string) ([]byte, error) {
	ctx := &interfaces.TransformContext{
		Content:     content,
		Filename:    filename,
		Diagnostics: nil,
		Metadata:    make(map[string]interface{}),
		DryRun:      false,
	}

	result, err := p.handler.Handle(ctx)
	if err != nil {
		return nil, err
	}

	if result.Diagnostics.HasErrors() {
		return nil, result.Diagnostics.Errs()[0]
	}

	return result.Content, nil
}

type PipelineBuilder struct {
	handlers []interfaces.TransformationHandler
	registry *registry.StrategyRegistry
}

func NewPipelineBuilder(reg *registry.StrategyRegistry) *PipelineBuilder {
	return &PipelineBuilder{
		handlers: make([]interfaces.TransformationHandler, 0),
		registry: reg,
	}
}

// With adds a handler using a factory function
func (b *PipelineBuilder) With(factory HandlerFactory) *PipelineBuilder {
	handler := factory(b.registry)
	b.handlers = append(b.handlers, handler)
	return b
}

// WithHandler adds a pre-created handler instance
func (b *PipelineBuilder) WithHandler(handler interfaces.TransformationHandler) *PipelineBuilder {
	b.handlers = append(b.handlers, handler)
	return b
}

// WithHandlers adds multiple pre-created handler instances
func (b *PipelineBuilder) WithHandlers(handlers ...interfaces.TransformationHandler) *PipelineBuilder {
	b.handlers = append(b.handlers, handlers...)
	return b
}

func (b *PipelineBuilder) Build() *Pipeline {
	if len(b.handlers) == 0 {
		return nil
	}

	for i := 0; i < len(b.handlers)-1; i++ {
		b.handlers[i].SetNext(b.handlers[i+1])
	}

	return &Pipeline{
		handler:  b.handlers[0],
		registry: b.registry,
	}
}
