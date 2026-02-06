package slimexporter

/*func TestNewFactory(t *testing.T) {
	factory := NewFactory()

	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}

	if factory.Type() != component.MustNewType(TypeStr) {
		t.Errorf("Type() = %v, want %v", factory.Type(), TypeStr)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	if cfg == nil {
		t.Fatal("CreateDefaultConfig() returned nil")
	}

	slimCfg, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("CreateDefaultConfig() returned wrong type: %T", cfg)
	}

	if slimCfg.SlimEndpoint != "http://127.0.0.1:46357" {
		t.Errorf("SlimEndpoint = %v, want http://127.0.0.1:46357", slimCfg.SlimEndpoint)
	}

	if slimCfg.ExporterNames.Metrics != "agntcy/otel/exporter-metrics" {
		t.Errorf("ExporterNames.Metrics = %v, want agntcy/otel/exporter-metrics", slimCfg.ExporterNames.Metrics)
	}

	if slimCfg.ExporterNames.Traces != "agntcy/otel/exporter-traces" {
		t.Errorf("ExporterNames.Traces = %v, want agntcy/otel/exporter-traces", slimCfg.ExporterNames.Traces)
	}

	if slimCfg.ExporterNames.Logs != "agntcy/otel/exporter-logs" {
		t.Errorf("ExporterNames.Logs = %v, want agntcy/otel/exporter-logs", slimCfg.ExporterNames.Logs)
	}
}
*/
