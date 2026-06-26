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

type ProfileInfo struct {
	Name         string
	Description  string
	Services     []string
	Routes       []string
	FailureModes []string
}

func Profiles() []ProfileInfo {
	return []ProfileInfo{
		{
			Name:        "web",
			Description: "HTTP request traces through a small service graph.",
			Services:    []string{"api-gateway", "catalog", "cart", "checkout", "payment", "inventory"},
			Routes:      []string{"GET /catalog", "GET /cart", "POST /checkout", "POST /payments", "GET /orders/:id"},
			FailureModes: []string{
				"HTTP 500 responses",
				"slow downstream calls",
				"retry attempts",
			},
		},
		{
			Name:        "grpc",
			Description: "gRPC service-to-service calls with RPC semantic attributes.",
			Services:    []string{"gateway", "catalog", "cart", "orders", "payments", "shipping"},
			Routes:      []string{"Gateway/GetCatalog", "Gateway/GetCart", "Gateway/PlaceOrder", "Gateway/AuthorizePayment"},
			FailureModes: []string{
				"synthetic RPC failures",
				"slow RPC calls",
				"retry events",
			},
		},
		{
			Name:        "queue",
			Description: "Asynchronous producer/consumer traces with messaging links.",
			Services:    []string{"api", "orders-worker", "payments-worker", "notifications-worker", "archive-worker"},
			Routes:      []string{"orders", "payments", "shipments", "returns", "invoices", "notifications"},
			FailureModes: []string{
				"producer failures",
				"consumer failures",
				"message processing retries",
			},
		},
		{
			Name:        "batch",
			Description: "Batch and scheduled job traces with chunked work.",
			Services:    []string{"scheduler", "worker", "postgres", "object-storage", "reporting"},
			Routes:      []string{"reindex-catalog", "daily-billing-rollup", "inventory-reconcile", "order-archive"},
			FailureModes: []string{
				"chunk failures",
				"slow batch steps",
				"retry attempts",
			},
		},
		{
			Name:        "payment-system",
			Description: "Checkout and refund traces through cart, pricing, fraud, payment, ledger, and email services.",
			Services:    []string{"edge-gateway", "checkout-api", "cart-service", "pricing-service", "fraud-service", "payment-service", "ledger-service", "email-service"},
			Routes:      []string{"POST /checkout", "POST /refund", "GET /orders/:id"},
			FailureModes: []string{
				"fraud service timeout",
				"payment provider 5xx",
				"ledger write retry",
				"email async failure",
				"duplicate idempotency key",
			},
		},
		{
			Name:        "api-gateway",
			Description: "Shallow high-volume gateway traces with auth, rate-limit, and upstream service calls.",
			Services:    []string{"gateway", "auth-service", "rate-limit-service", "catalog-api", "orders-api", "users-api", "search-api"},
			Routes:      []string{"GET /api/catalog", "GET /api/orders/:id", "POST /api/login", "GET /api/search", "POST /api/checkout"},
			FailureModes: []string{
				"auth failures",
				"rate-limited requests",
				"upstream 5xx responses",
				"route cardinality",
			},
		},
	}
}

func Profile(name string) (ProfileInfo, bool) {
	for _, p := range Profiles() {
		if p.Name == strings.ToLower(name) {
			return p, true
		}
	}
	return ProfileInfo{}, false
}

func moduleFor(name string) profileModule {
	switch strings.ToLower(name) {
	case "grpc":
		return grpcProfile{}
	case "queue":
		return queueProfile{}
	case "batch":
		return batchProfile{}
	case "payment-system":
		return paymentProfile{}
	case "api-gateway":
		return apiGatewayProfile{}
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

type paymentProfile struct{}

func (paymentProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	method, route := paymentOperation(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      method + " " + route,
		Kind:      "SERVER",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name":            "edge-gateway",
			"http.method":             method,
			"http.route":              route,
			"http.status_code":        200,
			"deployment.environment":  "checkout-lab",
			"payment.currency":        paymentCurrency(routeIdx),
			"payment.idempotency_key": "idem-synthetic",
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": "edge-gateway"}},
	}
}

func (paymentProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	target := paymentService(routeIdx)
	method, route := paymentOperation(routeIdx)
	attrs := model.Attrs{
		"service.name":            target,
		"peer.service":            target,
		"http.method":             method,
		"http.route":              route,
		"http.status_code":        200,
		"deployment.environment":  "checkout-lab",
		"payment.currency":        paymentCurrency(routeIdx),
		"payment.idempotency_key": "idem-synthetic",
	}
	switch target {
	case "fraud-service":
		attrs["fraud.score_bucket"] = fraudScoreBucket(routeIdx)
		attrs["failure.scenario"] = "fraud timeout"
	case "payment-service":
		attrs["payment.provider"] = paymentProvider(routeIdx)
		attrs["failure.scenario"] = "provider 5xx"
	case "ledger-service":
		attrs["ledger.account_type"] = ledgerAccountType(routeIdx)
		attrs["retry.count"] = 1
		attrs["failure.scenario"] = "ledger write retry"
	case "email-service":
		attrs["messaging.system"] = "smtp"
		attrs["failure.scenario"] = "async email failure"
	}
	addStoreAttrs(attrs, dbHeavy || target == "ledger-service", cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         paymentSpanName(target, routeIdx),
		Kind:         "CLIENT",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": target}},
	}
}

func (paymentProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
	return model.Span{}, model.Span{}
}

type apiGatewayProfile struct{}

func (apiGatewayProfile) buildRoot(frontdoor string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration) model.Span {
	method, route := gatewayOperation(routeIdx)
	return model.Span{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      method + " " + route,
		Kind:      "SERVER",
		StartTime: start,
		Duration:  dur,
		Attributes: model.Attrs{
			"service.name":       "gateway",
			"http.method":        method,
			"http.route":         route,
			"http.status_code":   gatewayStatus(routeIdx),
			"gateway.route_tier": gatewayRouteTier(routeIdx),
		},
		Status:   model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{"service.name": "gateway"}},
	}
}

func (apiGatewayProfile) buildChild(parent model.Span, service string, routeIdx int, start time.Time, spanID model.SpanID, traceID model.TraceID, dur time.Duration, dbHeavy bool, cacheHit bool) model.Span {
	target := gatewayService(routeIdx)
	method, route := gatewayOperation(routeIdx)
	status := gatewayStatus(routeIdx)
	attrs := model.Attrs{
		"service.name":       target,
		"peer.service":       target,
		"http.method":        method,
		"http.route":         route,
		"http.status_code":   status,
		"gateway.route_tier": gatewayRouteTier(routeIdx),
	}
	switch target {
	case "auth-service":
		attrs["auth.result"] = authResult(routeIdx)
		attrs["failure.scenario"] = "auth failure"
	case "rate-limit-service":
		attrs["rate_limit.decision"] = rateLimitDecision(routeIdx)
		attrs["failure.scenario"] = "rate limited"
	default:
		attrs["upstream.service"] = target
		if status >= 500 {
			attrs["failure.scenario"] = "upstream 5xx"
		}
	}
	addStoreAttrs(attrs, dbHeavy, cacheHit)
	return model.Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parent.SpanID,
		HasParent:    true,
		Name:         gatewaySpanName(target, routeIdx),
		Kind:         "CLIENT",
		StartTime:    start,
		Duration:     dur,
		Attributes:   attrs,
		Status:       model.SpanStatus{Code: "OK"},
		Resource:     model.Resource{Attributes: model.Attrs{"service.name": target}},
	}
}

func (apiGatewayProfile) buildQueuePair(parent model.Span, service string, routeIdx int, start time.Time, producerID model.SpanID, consumerID model.SpanID, traceID model.TraceID, dur time.Duration) (model.Span, model.Span) {
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

func paymentOperation(idx int) (string, string) {
	ops := []struct {
		method string
		route  string
	}{
		{method: "POST", route: "/checkout"},
		{method: "POST", route: "/refund"},
		{method: "GET", route: "/orders/:id"},
	}
	op := ops[idx%len(ops)]
	return op.method, op.route
}

func paymentService(idx int) string {
	services := []string{"checkout-api", "cart-service", "pricing-service", "fraud-service", "payment-service", "ledger-service", "email-service"}
	return services[idx%len(services)]
}

func paymentSpanName(service string, idx int) string {
	switch service {
	case "checkout-api":
		return "handle checkout"
	case "cart-service":
		return "load cart"
	case "pricing-service":
		return "calculate pricing"
	case "fraud-service":
		return "score fraud risk"
	case "payment-service":
		return "authorize payment"
	case "ledger-service":
		return "write ledger entry"
	case "email-service":
		return "send receipt"
	default:
		method, route := paymentOperation(idx)
		return method + " " + route
	}
}

func paymentCurrency(idx int) string {
	currencies := []string{"USD", "EUR", "GBP"}
	return currencies[idx%len(currencies)]
}

func paymentProvider(idx int) string {
	providers := []string{"stripe", "adyen", "checkout"}
	return providers[idx%len(providers)]
}

func fraudScoreBucket(idx int) string {
	buckets := []string{"low", "medium", "high"}
	return buckets[idx%len(buckets)]
}

func ledgerAccountType(idx int) string {
	types := []string{"customer", "merchant", "settlement"}
	return types[idx%len(types)]
}

func gatewayOperation(idx int) (string, string) {
	ops := []struct {
		method string
		route  string
	}{
		{method: "GET", route: "/api/catalog"},
		{method: "GET", route: "/api/orders/:id"},
		{method: "POST", route: "/api/login"},
		{method: "GET", route: "/api/search"},
		{method: "POST", route: "/api/checkout"},
		{method: "GET", route: "/api/users/:id"},
	}
	op := ops[idx%len(ops)]
	return op.method, op.route
}

func gatewayService(idx int) string {
	services := []string{"auth-service", "rate-limit-service", "catalog-api", "orders-api", "users-api", "search-api"}
	return services[idx%len(services)]
}

func gatewaySpanName(service string, idx int) string {
	switch service {
	case "auth-service":
		return "authorize request"
	case "rate-limit-service":
		return "check rate limit"
	default:
		method, route := gatewayOperation(idx)
		return method + " " + route
	}
}

func gatewayStatus(idx int) int {
	statuses := []int{200, 200, 401, 200, 429, 503}
	return statuses[idx%len(statuses)]
}

func gatewayRouteTier(idx int) string {
	tiers := []string{"public", "customer", "auth", "search", "checkout", "admin"}
	return tiers[idx%len(tiers)]
}

func authResult(idx int) string {
	if gatewayStatus(idx) == 401 {
		return "denied"
	}
	return "allowed"
}

func rateLimitDecision(idx int) string {
	if gatewayStatus(idx) == 429 {
		return "limited"
	}
	return "allowed"
}
