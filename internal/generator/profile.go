package generator

import (
	"fmt"
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

type profileModule interface {
	buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span
	buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span
	buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span)
}

func moduleFor(name string) profileModule {
	switch strings.ToLower(name) {
	case "grpc":
		return grpcProfile{}
	case "queue":
		return queueProfile{}
	case "batch":
		return batchProfile{}
	default:
		return webProfile{}
	}
}

type webProfile struct{}

func (webProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	method, route := webOperation(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      method + " " + route,
		Kind:      "SERVER",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name":     frontdoor,
			"http.method":      method,
			"http.route":       route,
			"http.status_code": 200,
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": frontdoor}},
	}
}

func (webProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	method, route := webOperation(routeIdx)
	attrs := model.Attrs{
		"service.name":     service,
		"peer.service":     service,
		"http.method":      method,
		"http.route":       route,
		"http.status_code": 200,
	}
	addStoreAttrs(attrs, dbHeavy, cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         method + " " + route,
		Kind:         "CLIENT",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
}

func (webProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
	return model.Span{}, model.Span{}
}

type grpcProfile struct{}

func (grpcProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	method := grpcMethod(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      "rpc Gateway/" + method,
		Kind:      "SERVER",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name": frontdoor,
			"rpc.system":   "grpc",
			"rpc.service":  "Gateway",
			"rpc.method":   method,
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": frontdoor}},
	}
}

func (grpcProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	method := grpcMethod(routeIdx)
	attrs := model.Attrs{
		"service.name": service,
		"peer.service": service,
		"rpc.system":   "grpc",
		"rpc.service":  fmt.Sprintf("%s.API", strings.ReplaceAll(service, "-", "")),
		"rpc.method":   method,
	}
	addStoreAttrs(attrs, dbHeavy, cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         "rpc " + service + "/" + method,
		Kind:         "CLIENT",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
}

func (grpcProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
	return model.Span{}, model.Span{}
}

type queueProfile struct{}

func (queueProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	topic := queueTopic(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      "enqueue " + topic,
		Kind:      "SERVER",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name":               frontdoor,
			"messaging.system":           "kafka",
			"messaging.destination.name": topic,
			"messaging.operation":        "publish",
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": frontdoor}},
	}
}

func (queueProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	attrs := model.Attrs{"service.name": service, "peer.service": service}
	addStoreAttrs(attrs, dbHeavy, cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         "handle work",
		Kind:         "INTERNAL",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
}

func (queueProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
	topic := queueTopic(routeIdx)
	producer := model.Span{
		TraceID:      traceID,
		SpanID:       producerID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         "publish " + topic,
		Kind:         "PRODUCER",
		StartTime:    start,
		Duration:     dur / 3,
		Attributes: model.Attrs{
			"service.name":               service,
			"messaging.system":           "kafka",
			"messaging.destination.name": topic,
			"messaging.operation":        "publish",
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
	consumer := model.Span{
		TraceID:      traceID,
		SpanID:       consumerID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         "consume " + topic,
		Kind:         "CONSUMER",
		StartTime:    start.Add(dur / 2),
		Duration:     dur / 2,
		Attributes: model.Attrs{
			"service.name":               service,
			"messaging.system":           "kafka",
			"messaging.destination.name": topic,
			"messaging.operation":        "process",
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
	producer.Links = append(producer.Links, model.Link{TraceID: traceID, SpanID: consumerID, Attributes: model.Attrs{"link.type": "follows_from"}})
	return producer, consumer
}

type batchProfile struct{}

func (batchProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	job := batchJob(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      "batch " + job,
		Kind:      "INTERNAL",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name": frontdoor,
			"batch.job":    job,
			"batch.run_id": "run-fixed",
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": frontdoor}},
	}
}

func (batchProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	job := batchJob(routeIdx)
	attrs := model.Attrs{
		"service.name": service,
		"batch.job":    job,
		"batch.chunk":  routeIdx,
	}
	addStoreAttrs(attrs, dbHeavy, cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         "batch step " + job,
		Kind:         "INTERNAL",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": service}},
	}
}

func (batchProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
	return model.Span{}, model.Span{}
}

func addStoreAttrs(attrs model.Attrs, dbHeavy bool, cacheHit bool) {
	if dbHeavy {
		attrs["db.system"] = "postgresql"
		attrs["db.operation"] = "SELECT"
	}
	attrs["cache.hit"] = cacheHit
}

func webOperation(idx int) (string, string) {
	type op struct {
		method string
		route  string
	}
	ops := []op{
		{method: "GET", route: "/catalog"},
		{method: "GET", route: "/cart"},
		{method: "POST", route: "/checkout"},
		{method: "POST", route: "/payments"},
		{method: "GET", route: "/orders/:id"},
		{method: "GET", route: "/inventory"},
		{method: "POST", route: "/login"},
		{method: "GET", route: "/search"},
	}
	p := ops[idx%len(ops)]
	return p.method, p.route
}

func grpcMethod(idx int) string {
	methods := []string{
		"GetCatalog",
		"GetCart",
		"PlaceOrder",
		"AuthorizePayment",
		"ReserveInventory",
		"ShipOrder",
		"ListOrders",
		"TrackOrder",
	}
	return methods[idx%len(methods)]
}

func queueTopic(idx int) string {
	topics := []string{"orders", "payments", "shipments", "returns", "invoices", "notifications"}
	return topics[idx%len(topics)]
}

func batchJob(idx int) string {
	jobs := []string{"reindex-catalog", "daily-billing-rollup", "inventory-reconcile", "order-archive", "sla-report", "fraud-retrain"}
	return jobs[idx%len(jobs)]
}
