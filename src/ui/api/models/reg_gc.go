// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego/validation"
	"github.com/vmware/harbor/src/common/job"
	"github.com/vmware/harbor/src/common/job/models"
	"github.com/vmware/harbor/src/common/utils"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/config"
)

const (
	//ScheduleDaily : 'Daily'
	ScheduleDaily = "Daily"
	//ScheduleWeekly : 'Weekly'
	ScheduleWeekly = "Weekly"
	//ScheduleManual : 'Manual'
	ScheduleManual = "Manual"
	//ScheduleNone : 'None'
	ScheduleNone = "None"
)

// GCReq holds request information for admin job
type GCReq struct {
	Schedule *ScheduleParam `json:"schedule"`
	Status   string         `json:"status"`
	ID       int64          `json:"id"`
}

//ScheduleParam defines the parameter of schedule trigger
type ScheduleParam struct {
	//Daily, Weekly, Manual, None
	Type string `json:"type"`
	//Optional, only used when type is 'weekly'
	Weekday int8 `json:"Weekday"`
	//The time offset with the UTC 00:00 in seconds
	Offtime int64 `json:"Offtime"`
}

// Valid validates the gc request
func (gr *GCReq) Valid(v *validation.Validation) {
	switch gr.Schedule.Type {
	case ScheduleDaily, ScheduleWeekly:
		if gr.Schedule.Offtime < 0 || gr.Schedule.Offtime > 3600*24 {
			v.SetError("offtime", fmt.Sprintf("Invalid schedule trigger parameter offtime: %d", gr.Schedule.Offtime))
		}
	case ScheduleManual, ScheduleNone:
	default:
		v.SetError("kind", fmt.Sprintf("Invalid schedule kind: %s", gr.Schedule.Type))
	}
}

// ToJob converts request to a job reconiged by job service.
func (gr *GCReq) ToJob() (*models.JobData, error) {
	metadata := &models.JobMetadata{
		JobKind: gr.JobKind(),
		// GC job must be unique ...
		IsUnique: true,
	}

	switch gr.Schedule.Type {
	case ScheduleDaily:
		h, m, s := utils.ParseOfftime(gr.Schedule.Offtime)
		metadata.Cron = fmt.Sprintf("%d %d %d * * *", s, m, h)
	case ScheduleWeekly:
		h, m, s := utils.ParseOfftime(gr.Schedule.Offtime)
		metadata.Cron = fmt.Sprintf("%d %d %d * * %d", s, m, h, gr.Schedule.Weekday%7)
	case ScheduleManual, ScheduleNone:
	default:
		return nil, fmt.Errorf("unsupported schedual trigger type: %s", gr.Schedule.Type)
	}

	jobData := &models.JobData{
		Name:     job.ImageGC,
		Metadata: metadata,
		StatusHook: fmt.Sprintf("%s/service/notifications/jobs/adminjob/%d",
			config.InternalUIURL(), gr.ID),
	}
	return jobData, nil
}

// IsPeriodic ...
func (gr *GCReq) IsPeriodic() bool {
	return gr.JobKind() == job.JobKindPeriodic
}

// JobKind ...
func (gr *GCReq) JobKind() string {
	switch gr.Schedule.Type {
	case ScheduleDaily, ScheduleWeekly:
		return job.JobKindPeriodic
	case ScheduleManual:
		return job.JobKindGeneric
	default:
		return ""
	}
}

// CronString ...
func (gr *GCReq) CronString() string {
	str, err := json.Marshal(gr.Schedule)
	if err != nil {
		log.Debugf("failed to marshal json error, %v", err)
		return ""
	}
	return string(str)
}