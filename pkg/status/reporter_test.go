package status

import (
	"bytes"
	"testing"
)

func TestReporter(t *testing.T) {
	buf := &bytes.Buffer{}
	indicator := NewIndicator(buf, true)
	reporter := NewReporter(indicator)

	// Test ReportSending
	reporter.ReportSending()
	if indicator.status != StatusSending {
		t.Errorf("expected status to be StatusSending, got %v", indicator.status)
	}

	// Test ReportSuccess
	reporter.ReportSuccess()
	if indicator.status != StatusSuccess {
		t.Errorf("expected status to be StatusSuccess, got %v", indicator.status)
	}

	// Test ReportFailure
	reporter.ReportFailure()
	if indicator.status != StatusFailed {
		t.Errorf("expected status to be StatusFailed, got %v", indicator.status)
	}
}

func TestReporterWithNilIndicator(t *testing.T) {
	reporter := NewReporter(nil)

	// Should not panic
	reporter.ReportSending()
	reporter.ReportSuccess()
	reporter.ReportFailure()
}
