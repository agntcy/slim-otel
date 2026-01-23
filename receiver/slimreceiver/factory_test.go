package slimreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()

	assert.NotNil(t, factory)
	assert.Equal(t, component.MustNewType(TypeStr), factory.Type())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	assert.NotNil(t, cfg)
	assert.IsType(t, &Config{}, cfg)

	receiverCfg := cfg.(*Config)
	assert.Equal(t, "http://127.0.0.1:46357", receiverCfg.SlimEndpoint)
	assert.Equal(t, "agntcy/otel/receiver", receiverCfg.ReceiverName)
	assert.Empty(t, receiverCfg.SharedSecret)
}
