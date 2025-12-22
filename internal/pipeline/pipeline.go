package pipeline

import (
	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type Pipeline struct {
	handler transform.TransformationHandler
	log     hclog.Logger
}

// BuildConfigPipeline creates the standard pipeline for HCL configuration files
// Pipeline: Preprocess → Parse → Transform → Format
func BuildConfigPipeline(log hclog.Logger, providers transform.MigrationProvider) *Pipeline {
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
// Pipeline: Preprocess → Transform → Format
func BuildStatePipeline(log hclog.Logger, providers transform.MigrationProvider) *Pipeline {
	preprocess := handlers.NewPreprocessHandler(providers)
	stateTransformer := handlers.NewStateTransformHandler(log, providers)
	format := handlers.NewStateFormatterHandler(log)

	// Chain handlers
	preprocess.SetNext(stateTransformer)
	stateTransformer.SetNext(format)

	return &Pipeline{
		handler: preprocess,
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
