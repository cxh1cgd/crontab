/**
*FileName: job
*Create on 2018-12-27 10:53
*Create by mok
 */

package job

import (
	"bufio"
	"context"
	"crontab/common/protocol"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var Executor *JobExecutor

type JobExecutor struct {
	m    sync.Map
	Done chan string
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

func InitExecutor() {
	Executor = &JobExecutor{
		Done: make(chan string, 50),
	}
}

func handleExecPlan(plan *JobPlan) {
	Executor.m.Store(plan.JobId, plan)
	defer func() {
		Executor.Done <- plan.JobId
	}()
	start := time.Now().UnixNano()
	lock := new(EtcdLock)
	lock.Key = protocol.JobLockDir + plan.JobId
	//没有获取到锁
	if lock.Lock() != nil {
		return
	}
	end := time.Now().UnixNano()
	fmt.Printf("cost:%d\n", end-start)
	plan.status = StatusExec
	result := execCmd(plan)
	lock.Unlock()
	plan.RestTime()

	//todo:异步保存任务结果
	result.JobId = plan.JobId
	result.Name = plan.Name
	//fmt.Println(result.OutPut)
}

func execCmd(plan *JobPlan) (result *protocol.JobResult) {
	var ctx, cancel = context.WithCancel(context.TODO())
	builder := strings.Builder{}
	result = new(protocol.JobResult)
	go func() {
		select {
		case <-plan.closed:
			cancel()
			builder.WriteString("任务已被关闭\n")
		}
	}()
	defer func() {
		now := time.Now()
		builder.WriteString(fmt.Sprintf("任务结束时间：%s,共计耗时:%s", now.String(), now.Sub(plan.nextTime).String()))
		result.OutPut = builder.String()
	}()
	builder.WriteString(fmt.Sprintf("任务计划开始时间：%s", plan.nextTime.String()))
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", plan.command)
	ppReader, err := cmd.StdoutPipe()
	defer ppReader.Close()
	if err != nil {
		builder.WriteString(fmt.Sprintf("任务启动失败，原因：%s", err.Error()))
		return result
	}
	err = cmd.Start()
	if err != nil {
		builder.WriteString(fmt.Sprintf("启动任务失败，开始时间：%s\n", time.Now().String()))
		return
	}
	builder.WriteString(fmt.Sprintf("启动任务成功，开始时间：%s\n", time.Now().String()))
	var bufReader = bufio.NewReader(ppReader)
	for {
		buffer := bufferPool.Get().([]byte)
		n, err := bufReader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				builder.WriteString("任务输出读取完成\n")
				break
			} else {
				builder.WriteString("任务输出读取失败\n")
			}
		}
		builder.Write(buffer[:n])
		bufferPool.Put(buffer)
	}
	cmd.Wait()
	builder.WriteString("任务执行成功\n")
	return result
}
