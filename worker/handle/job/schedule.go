/**
*FileName: job
*Create on 2018-12-25 10:21
*Create by mok
 */

package job

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const (
	MaxLevel = 32
)

var Scheduler *scheduler

type scheduler struct {
	eventChan chan *JobEvent
	table     *jobScheduleTable
	killChan  chan string
}

func InitScheduler() {
	Scheduler = &scheduler{
		eventChan: make(chan *JobEvent, 50),
	}
	initJobSchelduleTable()
	Scheduler.scheduleLoop()
}

func (s *scheduler) scheduleLoop() {
	go func() {
		for {
			select {
			//处理时间监听队列
			case e := <-s.eventChan:
				switch e.eventType {
				case TypeDelete:
					//todo:删除任务后要给该任务发送信号
					s.table.DeleteByJobId(e.job.JobId)
					if p, ok := Executor.m.Load(e.job.JobId); ok {
						plan := p.(*JobPlan)
						plan.closed <- struct{}{}
						Executor.m.Delete(plan.JobId)
					}
					log.Printf("删除任务成功，jobId:%s\n", e.job.JobId)
				//可能是更新也可能是新增任务
				case TypeUpdate:
					if plan := BuildJobPlan(e.job); plan != nil {
						if p, ok := Executor.m.Load(plan.JobId); ok {
							p.(*JobPlan).closed <- struct{}{}
							Executor.m.Delete(plan.JobId)
						}
						s.table.DeleteByJobId(plan.JobId)
						s.table.Insert(plan)
						log.Printf("更新任务列表成功，jobId:%s Name:%s\n", plan.JobId, plan.Name)
					}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			//强制杀死正在执行的任务
			case jobId := <-s.killChan:
				p, ok := Executor.m.Load(jobId)
				plan := p.(*JobPlan)
				if ok && plan.status == StatusExec {
					plan.closed <- struct{}{}
					plan.RestTime()
					Executor.m.Delete(jobId)
					s.table.Insert(plan)
				}
			}
			time.Sleep(30 * time.Millisecond)
		}
	}()

	//处理已经执行完毕的任务队列
	go func() {
		for {
			select {
			case jobId := <-Executor.Done:
				//有可能Done队列中有该任务id，但是有删除事件执行了，该plan在map中查询不到
				p, ok := Executor.m.Load(jobId)
				if ok {
					plan := p.(*JobPlan)
					Executor.m.Delete(jobId)
					s.table.Insert(plan)
				}
			}
		}
	}()

	//定期将要到期以及到期的任务取出
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		for {
			select {
			case <-ticker.C:

				if plans := s.table.Pop(); len(plans) != 0 {

					for _, plan := range plans {
						//执行任务
						go handleExecPlan(plan)
					}
				}
			}
		}
	}()
}

type jobNode struct {
	forward []*jobNode
	data    *JobPlan
}

type jobScheduleTable struct {
	head  *jobNode
	level int
}

func initJobSchelduleTable() {
	Scheduler.table = &jobScheduleTable{
		head:  &jobNode{forward: make([]*jobNode, MaxLevel)},
		level: 1,
	}
}

func newJobNode(data *JobPlan, level int) *jobNode {
	return &jobNode{forward: make([]*jobNode, level), data: data}
}

func (t *jobScheduleTable) Insert(data *JobPlan) {
	level := randLevel()
	if level > t.level {
		level = t.level + 1
		t.level = level
	}
	current := t.head
	before := make([]*jobNode, level)
	after := make([]*jobNode, level)
	for i := level; i >= 1; i-- {
		if current.forward[i-1] == nil || current.forward[i-1].data.nextTime.After(data.nextTime) {
			after[i-1] = current.forward[i-1]
			before[i-1] = current
		} else {
			for current.forward[i-1] != nil && (current.forward[i-1].data.nextTime.Before(data.nextTime) || current.forward[i-1].data.nextTime.Equal(data.nextTime)) {
				current = current.forward[i-1]
			}
			fmt.Println(i)
			before[i-1] = current
			after[i-1] = current.forward[i-1]
			/*for  {
				if current.forward[i-1] != nil{
					if current.forward[i-1].data.nextTime.Before(data.nextTime) {
						current = current.forward[i-1]
					}else {

					}
				}else {
					break
				}
			}*/
		}
	}

	node := newJobNode(data, level)
	for i := 0; i < level; i++ {
		node.forward[i] = after[i]
		before[i].forward[i] = node
	}
}

//通过jobId删除跳表中的任务
func (t *jobScheduleTable) DeleteByJobId(id string) {
	for i := t.level; i >= 1; i-- {
		current := t.head
		for current.forward[i-1] != nil {
			if current.forward[i-1].data.JobId == id {
				tmp := current.forward[i-1]
				current.forward[i-1] = tmp.forward[i-1]
				tmp.forward[i-1] = nil
			} else {
				current = current.forward[i-1]
			}
		}
	}
}

//弹出跳表中需要最近要执行的任务
func (t *jobScheduleTable) Pop() []*JobPlan {
	var plans = make([]*JobPlan, 0, 10000)
	for t.head.forward[0] != nil && len(plans) <= 10 {
		d := t.head.forward[0].data
		if d.nextTime.Before(time.Now()) || d.nextTime.Equal(time.Now()) || time.Now().Add(10*time.Millisecond).After(d.nextTime) {
			t.DeleteByJobId(d.JobId)
			plans = append(plans, d)
		} else {
			break
		}
	}

	return plans
}

func (t *jobScheduleTable) Print() {
	for i := t.level; i >= 1; i-- {
		current := t.head
		for {
			if current.forward[i-1] == nil {
				break
			}
			fmt.Printf("%s ", current.forward[i-1].data.JobId)
			current = current.forward[i-1]
		}
		fmt.Printf("***************** Level %d \n", i)
	}
}

func randLevel() int {
	rand.Seed(time.Now().UnixNano())
	var level = 1
	for {
		if rand.Float64() > 0.25 || level >= MaxLevel {
			break
		}
		level++
	}
	return level
}
