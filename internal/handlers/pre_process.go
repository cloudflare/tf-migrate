package handlers

import (
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type PreprocessHandler struct {
	transform.BaseHandler
	provider transform.MigrationProvider
}

func NewPreprocessHandler(provider transform.MigrationProvider) transform.TransformationHandler {
	return &PreprocessHandler{
		provider: provider,
	}
}

func (h *PreprocessHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	contentStr := string(ctx.Content)
	contentStr = h.applyAllPreprocessors(ctx, contentStr)
	ctx.Content = []byte(contentStr)
	return h.Next(ctx)
}

func (h *PreprocessHandler) applyAllPreprocessors(ctx *transform.Context, content string) string {
	for _, migrator := range h.provider.GetAllMigrators(ctx.SourceVersion, ctx.TargetVersion, ctx.Resources...) {
		content = migrator.Preprocess(content)
	}
	return content
}
