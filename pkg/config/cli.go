package config

type CommonConfig struct {
	BaseLogDir       string `help:"Base log directory" env:"LLM_REDACTOR_LOG_DIR" default:"~/.llm-redactor"`
	AppLogFile       string `help:"Application log file" env:"LLM_REDACTOR_APP_LOG_FILE" default:"app.jsonl"`
	TrafficLogFile   string `help:"Traffic log file" env:"LLM_REDACTOR_TRAFFIC_LOG_FILE" default:"traffic.jsonl"`
	DetectionLogFile string `help:"Detection log file" env:"LLM_REDACTOR_DETECTION_LOG_FILE" default:"detections.jsonl"`
	RedactorRules    string `help:"Redactor rules file (TOML or JSON)" env:"LLM_REDACTOR_REDACTOR_RULES" default:"~/.gitleaks.toml"`
	Version          bool   `help:"Show version information" short:"v"`
}

type ExecCLI struct {
	CommonConfig
	Host    string   `help:"Host" env:"LLM_REDACTOR_HOST" default:"127.0.0.1"`
	Port    int      `help:"Port" env:"LLM_REDACTOR_PORT" default:"0"`
	Command []string `arg:"" optional:"" help:"Command to execute"`
}

type ProxyCLI struct {
	CommonConfig
	Host string `help:"Host" env:"LLM_REDACTOR_HOST" default:"0.0.0.0"`
	Port int    `help:"Port" env:"LLM_REDACTOR_PORT" default:"4000"`
}
