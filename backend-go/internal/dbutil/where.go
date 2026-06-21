package dbutil

import (
	"fmt"
	"strings"
)

// WhereBuilder keeps SQL WHERE conditions and positional arguments in sync.
// The condition format must contain one %d placeholder for the next $n marker.
type WhereBuilder struct {
	conditions []string
	args       []any
}

// NewWhereBuilder starts a WHERE builder with fixed conditions that do not
// consume query arguments.
func NewWhereBuilder(conditions ...string) *WhereBuilder {
	return &WhereBuilder{conditions: append([]string(nil), conditions...)}
}

// Add appends value as the next positional argument and adds the matching
// condition, for example Add("owner_user_id = $%d", ownerID).
func (b *WhereBuilder) Add(conditionFormat string, value any) {
	if strings.Count(conditionFormat, "%d") != 1 {
		panic(fmt.Sprintf("dbutil.WhereBuilder.Add condition format must contain exactly one %%d placeholder: %q", conditionFormat))
	}
	b.args = append(b.args, value)
	b.conditions = append(b.conditions, fmt.Sprintf(conditionFormat, len(b.args)))
}

// SQL returns the joined WHERE condition body.
func (b *WhereBuilder) SQL() string {
	return strings.Join(b.conditions, " AND ")
}

// Args returns the positional query arguments.
func (b *WhereBuilder) Args() []any {
	return append([]any(nil), b.args...)
}
