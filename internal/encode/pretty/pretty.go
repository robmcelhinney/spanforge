package pretty

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func RenderTrace(trace model.Trace) string {
	var b strings.Builder
	traceID := hex.EncodeToString(trace.TraceID[:])
	b.WriteString("trace ")
	b.WriteString(traceID)
	b.WriteByte('\n')

	children := map[string][]model.Span{}
	roots := make([]model.Span, 0, len(trace.Spans))
	for _, span := range trace.Spans {
		if span.HasParent {
			k := hex.EncodeToString(span.ParentSpanID[:])
			children[k] = append(children[k], span)
		} else {
			roots = append(roots, span)
		}
	}

	sort.Slice(roots, func(i, j int) bool { return roots[i].StartTime.Before(roots[j].StartTime) })
	for _, r := range roots {
		renderSpan(&b, r, children, 0)
	}
	return b.String()
}

func renderSpan(b *strings.Builder, span model.Span, children map[string][]model.Span, level int) {
	for i := 0; i < level; i++ {
		b.WriteString("  ")
	}
	service, _ := span.Attributes["service.name"].(string)
	fmt.Fprintf(b, "- %s [%s] (%s, %0.2fms, %s)\n", span.Name, span.Kind, service, float64(span.Duration)/1e6, span.Status.Code)

	k := hex.EncodeToString(span.SpanID[:])
	next := children[k]
	sort.Slice(next, func(i, j int) bool { return next[i].StartTime.Before(next[j].StartTime) })
	for _, c := range next {
		renderSpan(b, c, children, level+1)
	}
}
