/**
*FileName: protocol
*Create on 2018-12-18 17:19
*Create by mok
 */

package protocol

import (
	"sync"
)

const (
	JobSaveDir = "/jobs/exec/"
	JobKillDir = "/jobs/kill/"
	JobLockDir = "/jobs/lock/"
)

var (
	JobPool = sync.Pool{
		New: func() interface{} {
			return &Job{}
		},
	}
)

type Job struct {
	JobId      string `json:"job_id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	Expression string `json:"expression"`
	Times      int    `json:"times"`
}

type JobResult struct {
	OutPut string //执行输出结果
	JobId  string
	Name   string
}
