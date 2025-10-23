package pipeline

import (
	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/go-hclog"
)

type Pipeline struct {
	handler transform.TransformationHandler
	log     hclog.Logger
}

// BuildConfigPipeline creates the standard pipeline for HCL configuration files
// Pipeline: Preprocess → Parse → Transform → Format
func BuildConfigPipeline(log hclog.Logger, providers transform.Provider) *Pipeline {
	preprocess := handlers.NewPreprocessHandler(providers)
	parse := handlers.NewParseHandler(log)
	resourceTransformer := handlers.NewResourceTransformHandler(log, providers)
	format := handlers.NewFormatterHandler(log)

	// Chain handlers
	preprocess.SetNext(parse)
	parse.SetNext(resourceTransformer)
	resourceTransformer.SetNext(format)

	return &Pipeline{
		handler: preprocess,
		log:     log,
	}
}

// BuildStatePipeline creates the standard pipeline for JSON state files
// Pipeline: Transform → Format
func BuildStatePipeline(log hclog.Logger, providers transform.Provider) *Pipeline {
	stateTransformer := handlers.NewStateTransformHandler(log, providers)
	format := handlers.NewStateFormatterHandler(log)

	// Chain handlers
	stateTransformer.SetNext(format)

	return &Pipeline{
		handler: stateTransformer,
		log:     log,
	}
}

// Transform executes the pipeline on the given content
func (p *Pipeline) Transform(ctx *transform.Context) ([]byte, error) {
	result, err := p.handler.Handle(ctx)
	if err != nil {
		return nil, err
	}

	if result.Diagnostics.HasErrors() {
		return nil, result.Diagnostics.Errs()[0]
	}

	return result.Content, nil
}
