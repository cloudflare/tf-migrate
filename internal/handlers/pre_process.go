package handlers

import (
	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

type PreprocessHandler struct {
	interfaces.BaseHandler
	registry *registry.StrategyRegistry
}

func NewPreprocessHandler(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
	return &PreprocessHandler{
		registry: reg,
	}
}

func (h *PreprocessHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	contentStr := string(ctx.Content)
	contentStr = h.applyAllPreprocessors(contentStr)
	ctx.Content = []byte(contentStr)
	return h.CallNext(ctx)
}

func (h *PreprocessHandler) applyAllPreprocessors(content string) string {
	strategies := h.registry.GetAll()

	for _, strategy := range strategies {
		content = strategy.Preprocess(content)
	}

	return content
}
