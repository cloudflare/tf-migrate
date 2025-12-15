package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetStateTypeRename(t *testing.T) {
	t.Run("initializes StateTypeRenames map when nil", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "example", "cloudflare_argo", "cloudflare_argo_smart_routing")

		assert.NotNil(t, ctx.StateTypeRenames)
		assert.Equal(t, "cloudflare_argo_smart_routing", ctx.StateTypeRenames["cloudflare_argo.example"])
	})

	t.Run("adds to existing StateTypeRenames map", func(t *testing.T) {
		ctx := &Context{
			StateTypeRenames: map[string]interface{}{
				"existing.key": "existing_value",
			},
		}

		SetStateTypeRename(ctx, "example", "cloudflare_argo", "cloudflare_argo_smart_routing")

		assert.Len(t, ctx.StateTypeRenames, 2)
		assert.Equal(t, "existing_value", ctx.StateTypeRenames["existing.key"])
		assert.Equal(t, "cloudflare_argo_smart_routing", ctx.StateTypeRenames["cloudflare_argo.example"])
	})

	t.Run("overwrites existing entry", func(t *testing.T) {
		ctx := &Context{
			StateTypeRenames: map[string]interface{}{
				"cloudflare_argo.example": "cloudflare_argo_smart_routing",
			},
		}

		SetStateTypeRename(ctx, "example", "cloudflare_argo", "cloudflare_argo_tiered_caching")

		assert.Len(t, ctx.StateTypeRenames, 1)
		assert.Equal(t, "cloudflare_argo_tiered_caching", ctx.StateTypeRenames["cloudflare_argo.example"])
	})

	t.Run("handles multiple different resources", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "example1", "cloudflare_argo", "cloudflare_argo_smart_routing")
		SetStateTypeRename(ctx, "example2", "cloudflare_argo", "cloudflare_argo_tiered_caching")
		SetStateTypeRename(ctx, "generic", "cloudflare_tiered_cache", "cloudflare_argo_tiered_caching")

		assert.Len(t, ctx.StateTypeRenames, 3)
		assert.Equal(t, "cloudflare_argo_smart_routing", ctx.StateTypeRenames["cloudflare_argo.example1"])
		assert.Equal(t, "cloudflare_argo_tiered_caching", ctx.StateTypeRenames["cloudflare_argo.example2"])
		assert.Equal(t, "cloudflare_argo_tiered_caching", ctx.StateTypeRenames["cloudflare_tiered_cache.generic"])
	})

	t.Run("creates correct key format", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "my_resource", "cloudflare_example", "cloudflare_new_example")

		_, exists := ctx.StateTypeRenames["cloudflare_example.my_resource"]
		assert.True(t, exists, "Key should be in format 'originalType.resourceName'")
	})

	t.Run("handles empty resource name", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "", "cloudflare_argo", "cloudflare_argo_smart_routing")

		assert.Equal(t, "cloudflare_argo_smart_routing", ctx.StateTypeRenames["cloudflare_argo."])
	})

	t.Run("handles empty original type", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "example", "", "cloudflare_argo_smart_routing")

		assert.Equal(t, "cloudflare_argo_smart_routing", ctx.StateTypeRenames[".example"])
	})

	t.Run("handles empty target type", func(t *testing.T) {
		ctx := &Context{}

		SetStateTypeRename(ctx, "example", "cloudflare_argo", "")

		assert.Equal(t, "", ctx.StateTypeRenames["cloudflare_argo.example"])
	})
}
