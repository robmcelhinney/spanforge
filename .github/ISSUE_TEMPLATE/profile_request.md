---
name: Profile request
about: Suggest a realistic workload for spanforge to generate
title: "profile: "
labels: enhancement
assignees: ""
---

## User need

Describe the tracing pipeline or backend test that needs this profile.

## Workload

Describe the real system or traffic pattern that the profile should represent.

## Services and operations

List the important services, routes, jobs, database calls or messages.

## Trace shape

Describe the parent and child spans, links, events and attributes that matter.

## Failure modes

List the errors, retries, latency changes or traffic changes that the profile should include.

## Why existing profiles do not work

Explain why `payment-system`, `api-gateway`, `web`, `grpc`, `queue` or `batch` cannot model this need.
