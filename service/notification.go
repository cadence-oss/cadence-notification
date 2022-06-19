package service

import (
	"time"

	"github.com/uber/cadence/common"
)

type (
	Notification struct {
		ID                  string
		VisibilityOperation common.VisibilityOperation
		// TODO: replace with domainName, need to pass by Cadence server
		DomainID         string
		WorkflowID       string
		RunID            string
		WorkflowType     string
		StartedTimestamp time.Time
		// the actual time that starting execution, this is used mainly for cron schedule workflow
		ExecutionTimestamp time.Time
		ClosedTimestamp    time.Time
		SearchAttributes   map[string]interface{}
		Memo               map[string]interface{}
	}
)
