// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// This file contains code copied from the OpenTelemetry Go project:
// https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/otlp/otlptrace/internal/tracetransform/resource.go
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"

	"go.opentelemetry.io/otel/sdk/resource"
)

// Resource transforms a Resource into an OTLP Resource.
func Resource(r *resource.Resource) *resourcepb.Resource {
	if r == nil {
		return nil
	}
	return &resourcepb.Resource{Attributes: ResourceAttributes(r)}
}
