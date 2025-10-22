package transform

// TransformationHandler defines the interface for pipeline handlers
type TransformationHandler interface {
	Handle(ctx *Context) (*Context, error)
	SetNext(handler TransformationHandler)
	Next(ctx *Context) (*Context, error)
}

// BaseHandler provides common functionality for pipeline handlers
type BaseHandler struct {
	next TransformationHandler
}

func (h *BaseHandler) SetNext(handler TransformationHandler) {
	h.next = handler
}

func (h *BaseHandler) Next(ctx *Context) (*Context, error) {
	if h.next != nil {
		return h.next.Handle(ctx)
	}
	return ctx, nil
}
