package service

import "time"

type (
	Notification struct {
		ID               string
		NotificationType NotificationTypeEnum
		// TODO: replace with domainName, need to pass by Cadence server
		DomainID         string
		WorkflowID       string
		RunID            string
		WorkflowType     string
		StartedTimestamp time.Time
		ClosedTimestamp  time.Time
		SearchAttributes map[string]interface{}
		Memo             map[string]interface{}
	}

	NotificationTypeEnum string
)

const (
	NotificationTypeWorkflowStarted NotificationTypeEnum = "workflowStarted"
	NotificationTypeWorkflowClosed NotificationTypeEnum = "workflowClosed"
	NotificationTypeWorkflowUpsertSearchAttributes NotificationTypeEnum = "upsertSearchAttributes"
)