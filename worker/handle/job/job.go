/**
*FileName: job
*Create on 2018-12-25 10:21
*Create by mok
 */

package job

import (
	"crontab/common/protocol"
	"fmt"
	"github.com/gorhill/cronexpr"
	"sync"
	"time"
)

type EventType int

const (
	TypeCreate EventType = iota
	TypeUpdate
	TypeDelete
	StatusSleep int = 0
	StatusExec  int = 1
)

var eventPool = &sync.Pool{
	New: func() interface{} {
		return &JobEvent{
			job: protocol.JobPool.Get().(*protocol.Job),
		}
	},
}

type JobEvent struct {
	eventType EventType
	job       *protocol.Job
}

type JobPlan struct {
	JobId    string
	Name     string
	nextTime time.Time
	expr     *cronexpr.Expression
	command  string
	status   int
	closed   chan struct{}
}

func BuildJobPlan(job *protocol.Job) *JobPlan {
	var jp = new(JobPlan)
	jp.command = job.Command
	jp.JobId = job.JobId
	jp.Name = job.Name
	jp.status = StatusSleep
	exp, err := cronexpr.Parse(job.Expression)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	jp.expr = exp
	jp.nextTime = exp.Next(time.Now())
	return jp
}

func (plan *JobPlan) RestTime() {
	plan.status = StatusSleep
	plan.nextTime = plan.expr.Next(time.Now())
}
