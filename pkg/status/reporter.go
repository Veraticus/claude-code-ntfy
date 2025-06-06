package status

import "github.com/Veraticus/claude-code-ntfy/pkg/interfaces"

// Reporter adapts the Indicator to implement interfaces.StatusReporter
type Reporter struct {
	indicator *Indicator
}

// NewReporter creates a new status reporter
func NewReporter(indicator *Indicator) *Reporter {
	return &Reporter{
		indicator: indicator,
	}
}

// Ensure Reporter implements StatusReporter
var _ interfaces.StatusReporter = (*Reporter)(nil)

// ReportSending reports that a notification is being sent
func (r *Reporter) ReportSending() {
	if r.indicator != nil {
		r.indicator.SetStatus(StatusSending)
	}
}

// ReportSuccess reports that a notification was sent successfully
func (r *Reporter) ReportSuccess() {
	if r.indicator != nil {
		r.indicator.SetStatus(StatusSuccess)
	}
}

// ReportFailure reports that a notification failed to send
func (r *Reporter) ReportFailure() {
	if r.indicator != nil {
		r.indicator.SetStatus(StatusFailed)
	}
}
