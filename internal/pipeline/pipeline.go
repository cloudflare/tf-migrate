package pipeline

import (
	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type Pipeline struct {
	handler       transform.TransformationHandler
	log           hclog.Logger
	sourceVersion string
	targetVersion string
}

// BuildConfigPipeline creates the standard pipeline for HCL configuration files
// Pipeline: Preprocess → Parse → Transform → Format
func BuildConfigPipeline(log hclog.Logger, sourceVersion, targetVersion string) *Pipeline {
	// Create provider with version-aware functions
	getFunc := func(resourceType string, source string, target string) transform.ResourceTransformer {
		return internal.GetMigrator(resourceType, source, target)
	}
	getAllFunc := func(source string, target string) []transform.ResourceTransformer {
		return internal.GetAllMigrators(source, target)
	}
	providers := transform.NewMigratorProvider(getFunc, getAllFunc)
	preprocess := handlers.NewPreprocessHandler(providers)
	parse := handlers.NewParseHandler(log)
	resourceTransformer := handlers.NewResourceTransformHandler(log, providers)
	format := handlers.NewFormatterHandler(log)

	// Chain handlers
	preprocess.SetNext(parse)
	parse.SetNext(resourceTransformer)
	resourceTransformer.SetNext(format)

	return &Pipeline{
		handler:       preprocess,
		log:           log,
		sourceVersion: sourceVersion,
		targetVersion: targetVersion,
	}
}

// BuildStatePipeline creates the standard pipeline for JSON state files
// Pipeline: Transform → Format
func BuildStatePipeline(log hclog.Logger, sourceVersion, targetVersion string) *Pipeline {
	// Create provider with version-aware functions
	getFunc := func(resourceType string, source string, target string) transform.ResourceTransformer {
		return internal.GetMigrator(resourceType, source, target)
	}
	getAllFunc := func(source string, target string) []transform.ResourceTransformer {
		return internal.GetAllMigrators(source, target)
	}
	providers := transform.NewMigratorProvider(getFunc, getAllFunc)
	stateTransformer := handlers.NewStateTransformHandler(log, providers)
	format := handlers.NewStateFormatterHandler(log)

	// Chain handlers
	stateTransformer.SetNext(format)

	return &Pipeline{
		handler:       stateTransformer,
		log:           log,
		sourceVersion: sourceVersion,
		targetVersion: targetVersion,
	}
}

// Transform executes the pipeline on the given content
func (p *Pipeline) Transform(content []byte, filename string) ([]byte, error) {
	ctx := &transform.Context{
		Content:       content,
		Filename:      filename,
		Diagnostics:   nil,
		Metadata:      make(map[string]interface{}),
		DryRun:        false,
		SourceVersion: p.sourceVersion,
		TargetVersion: p.targetVersion,
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
