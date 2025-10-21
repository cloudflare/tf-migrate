package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/vaishak/tf-migrate/internal/interfaces"
)

type StateFormatterHandler struct {
	interfaces.BaseHandler
}

func NewStateFormatterHandler() interfaces.TransformationHandler {
	return &StateFormatterHandler{}
}

func (h *StateFormatterHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	if len(ctx.Content) == 0 {
		return ctx, fmt.Errorf("state content is empty - cannot format")
	}
	var formatted interface{}

	if err := json.Unmarshal(ctx.Content, &formatted); err != nil {
		// If we can't parse it, leave it as is
		return h.CallNext(ctx)
	}

	prettyJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		// If we can't format it, leave it as is
		return h.CallNext(ctx)
	}

	ctx.Content = prettyJSON
	return h.CallNext(ctx)
}
