package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

type StateFormatterHandler struct {
	transform.BaseHandler
	log hclog.Logger
}

func NewStateFormatterHandler(log hclog.Logger) transform.TransformationHandler {
	return &StateFormatterHandler{
		log: log,
	}
}

func (h *StateFormatterHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	if len(ctx.Content) == 0 {
		return ctx, fmt.Errorf("state content is empty - cannot format")
	}
	var formatted interface{}

	if err := json.Unmarshal(ctx.Content, &formatted); err != nil {
		// If we can't parse it, leave it as is
		return h.Next(ctx)
	}

	prettyJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		// If we can't format it, leave it as is
		return h.Next(ctx)
	}

	ctx.Content = prettyJSON
	return h.Next(ctx)
}
