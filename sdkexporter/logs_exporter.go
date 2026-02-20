// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"sync"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/logtransform"
)

// LogExporter exports logs to SLIM
type LogExporter struct {
	connID     uint64
	app        *slim.App
	sessions   *slimcommon.SessionsList
	mu         sync.RWMutex
	stopped    bool
	cancelFunc context.CancelFunc
}

// Export exports log records to SLIM
// This implements the sdklog.Exporter interface
func (le *LogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if le.stopped {
		return nil
	}

	if len(records) == 0 {
		return nil
	}

	// Convert SDK log records to OTLP protobuf ResourceLogs format
	resourceLogs := logtransform.ResourceLogs(records)
	if len(resourceLogs) == 0 {
		return nil
	}

	// Create OTLP ExportLogsServiceRequest with all ResourceLogs
	req := &collectorlogs.ExportLogsServiceRequest{
		ResourceLogs: resourceLogs,
	}

	// Marshal to protobuf bytes
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal logs request: %w", err)
	}

	// Publish to all logs sessions
	closedSessions, err := le.sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			le.sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// ForceFlush flushes any pending logs
// This implements the sdklog.Exporter interface
func (le *LogExporter) ForceFlush(ctx context.Context) error {
	// SLIM publishes immediately, no buffering to flush
	return nil
}

// Shutdown shuts down the log exporter
// This implements the sdklog.Exporter interface
func (le *LogExporter) Shutdown(ctx context.Context) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if le.stopped {
		return nil
	}
	le.stopped = true

	// Stop the session listener
	if le.cancelFunc != nil {
		le.cancelFunc()
	}

	// Remove all sessions
	le.sessions.DeleteAll(ctx, le.app)
	// Destroy the app
	le.app.Destroy()

	return nil
}
