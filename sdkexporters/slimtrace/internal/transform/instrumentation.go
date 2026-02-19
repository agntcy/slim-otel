// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// This file contains code copied from the OpenTelemetry Go project:
// https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/otlp/otlptrace/internal/tracetransform/instrumentation.go
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"

	"go.opentelemetry.io/otel/sdk/instrumentation"
)

func InstrumentationScope(il instrumentation.Scope) *commonpb.InstrumentationScope {
	if il == (instrumentation.Scope{}) {
		return nil
	}
	return &commonpb.InstrumentationScope{
		Name:       il.Name,
		Version:    il.Version,
		Attributes: Iterator(il.Attributes.Iter()),
	}
}
