package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BenchmarkStructuredLog 基准测试结构化日志 (无脱敏)
func BenchmarkStructuredLog(b *testing.B) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message",
			zap.String("key1", "value1"),
			zap.Int("key2", 42),
			zap.String("workflow_id", "wf-123"),
		)
	}
}

// BenchmarkStructuredLogWithSanitization 基准测试带脱敏的结构化日志
func BenchmarkStructuredLogWithSanitization(b *testing.B) {
	zapCfg := zap.NewProductionConfig()
	zapCfg.OutputPaths = []string{"/dev/null"} // 避免I/O影响性能

	logger, _ := zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return NewSanitizingCore(core, DefaultSensitiveFields)
	}))
	defer logger.Sync()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message",
			zap.String("key1", "value1"),
			zap.Int("key2", 42),
			zap.String("workflow_id", "wf-123"),
			zap.String("password", "secret"), // 敏感字段
		)
	}
}

// BenchmarkFormattedLog 基准测试格式化日志 (Sugar API)
func BenchmarkFormattedLog(b *testing.B) {
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	defer logger.Sync()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sugar.Infof("Benchmark message: %s, %d", "value1", 42)
	}
}

// BenchmarkWithContext 基准测试上下文 logger 创建
func BenchmarkWithContext(b *testing.B) {
	err := Init("info", "json")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = WithWorkflowContext("wf-123")
	}
}

// BenchmarkCheckedEntry 基准测试 CheckedEntry (避免无效日志构造)
func BenchmarkCheckedEntry(b *testing.B) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if ce := logger.Check(zap.DebugLevel, "Debug message"); ce != nil {
			ce.Write(zap.String("key", "value"))
		}
	}
}
