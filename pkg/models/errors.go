package models

import "fmt"

// ErrorDetail represents detailed error information from the Edge Server
type ErrorDetail struct {
	Message  string        `json:"message"`
	Code     string        `json:"code"`
	Inner    string        `json:"inner"`
	Location ErrorLocation `json:"location"`
}

// ErrorLocation provides context about where an error occurred
type ErrorLocation struct {
	Namespace           string `json:"namespace"`
	WorkflowName        string `json:"workflowName"`
	ProcessorName       string `json:"processorName"`
	ProcessorType       string `json:"processorType"`
	ProcessorConfigName string `json:"processorConfigName"`
	ProcessId           string `json:"processId"`
}

// Error implements the error interface for ErrorDetail
func (e *ErrorDetail) Error() string {
	if e.Inner == "" {
		return fmt.Sprintf("%s: %s (at namespace=%s, workflowName=%s, processorName=%s, processorType=%s)",
			e.Code, e.Message, e.Location.Namespace, e.Location.WorkflowName,
			e.Location.ProcessorName, e.Location.ProcessorType)
	}
	return fmt.Sprintf("%s: %s (at namespace=%s, workflowName=%s, processorName=%s, processorType=%s) - caused by: %s",
		e.Code, e.Message, e.Location.Namespace, e.Location.WorkflowName,
		e.Location.ProcessorName, e.Location.ProcessorType, e.Inner)
}
