# E2E-SKIP: for expression items cannot be tested in automated E2E runs
#
# REASON FOR SKIP:
# ================
# In v4, `items` is list(string), so `items = [for k, v in map : k]` is valid.
# In v5, `items` is list(object({value, description})), so a for expression
# producing plain strings is type-incompatible with the v5 schema. The resource
# cannot be applied under v5 after migration without rewriting the expression —
# which tf-migrate intentionally leaves untouched (opaque).
#
# The main zero_trust_list.tf already covers all other patterns for E2E.
#
# TESTING COVERAGE:
# =================
# ✓ Unit tests: internal/resources/zero_trust_list/v4_to_v5_test.go
#   - TestV4ToV5Transformation_ForExpression (4 cases)
#
# ✓ Array unit tests: internal/transform/hcl/arrays_test.go
#   - TestForExpressionHandling (3 cases)
#   - TestParseArrayAttributeForExpression (3 cases)
#
# ✓ Integration tests: integration/v4_to_v5/testdata/zero_trust_list/
#   - Confirms for expressions are preserved verbatim in migrated output
#
# ✗ E2E tests: v5 schema requires object form; plain-string for expressions
#   are type-incompatible with the v5 `items` attribute after migration.
