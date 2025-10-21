package interfaces

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type TransformContext struct {
	Content     []byte
	Filename    string
	AST         *hclwrite.File
	StateJSON   string
	Diagnostics hcl.Diagnostics
	Metadata    map[string]interface{}
	DryRun      bool
}

type TransformationHandler interface {
	Handle(ctx *TransformContext) (*TransformContext, error)
	SetNext(handler TransformationHandler)
}

type BaseHandler struct {
	next TransformationHandler
}

func (h *BaseHandler) SetNext(handler TransformationHandler) {
	h.next = handler
}

func (h *BaseHandler) CallNext(ctx *TransformContext) (*TransformContext, error) {
	if h.next != nil {
		return h.next.Handle(ctx)
	}
	return ctx, nil
}