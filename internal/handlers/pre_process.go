package handlers

import (
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type PreprocessHandler struct {
	transform.BaseHandler
	provider transform.Provider
}

func NewPreprocessHandler(provider transform.Provider) transform.TransformationHandler {
	return &PreprocessHandler{
		provider: provider,
	}
}

func (h *PreprocessHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	contentStr := string(ctx.Content)
	contentStr = h.applyAllPreprocessors(contentStr)
	ctx.Content = []byte(contentStr)
	return h.Next(ctx)
}

func (h *PreprocessHandler) applyAllPreprocessors(content string) string {
	for _, migrator := range h.provider.GetAllMigrators() {
		content = migrator.Preprocess(content)
	}
	return content
}
