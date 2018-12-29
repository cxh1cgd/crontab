/**
*FileName: job
*Create on 2018-12-25 10:21
*Create by mok
 */

package job

import (
	"context"
	"crontab/common/protocol"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"log"
	"strings"
)

var Manager *jobManager

type jobManager struct {
	client *clientv3.Client
}

func InitManager() {
	Manager = &jobManager{
		client: Client,
	}
	go Manager.manageJobs()
}

func (manager *jobManager) manageJobs() {
	resp, err := manager.client.Get(context.TODO(), protocol.JobSaveDir, clientv3.WithPrefix())
	if err != nil {
		log.Fatalf("任务管理器获取任务列表失败，程序即将关闭")
	}
	for _, pair := range resp.Kvs {
		job := protocol.JobPool.Get().(*protocol.Job)
		if err = json.Unmarshal(pair.Value, job); err != nil {
			log.Println(err.Error())
			continue
		}
		if plan := BuildJobPlan(job); plan != nil {
			Scheduler.table.Insert(plan)
		}
	}

	//任务目录监听
	go func() {
		wc := manager.client.Watch(context.TODO(), protocol.JobSaveDir, clientv3.WithPrefix())
		for {
			select {
			case wcResp := <-wc:
				for _, event := range wcResp.Events {
					jobEvent := eventPool.Get().(*JobEvent)
					switch event.Type {
					case mvccpb.DELETE:
						fmt.Println("删除事件")
						jobEvent.eventType = TypeDelete
						jobEvent.job.JobId = strings.TrimPrefix(string(event.Kv.Key), "/jobs/exec/")
					case mvccpb.PUT:
						if err := json.Unmarshal(event.Kv.Value, jobEvent.job); err != nil {
							log.Println(err.Error())
							continue
						}
						jobEvent.eventType = TypeUpdate
					}
					Scheduler.eventChan <- jobEvent
					eventPool.Put(jobEvent)
				}
			}
		}

	}()

	//监听强杀目录
	go func() {
		wc := manager.client.Watch(context.TODO(), protocol.JobKillDir, clientv3.WithPrefix())
		for {
			select {
			case wcResp := <-wc:
				for _, event := range wcResp.Events {
					if event.Type == mvccpb.PUT && event.IsCreate() {
						jobId := string(event.Kv.Key)
						Scheduler.killChan <- jobId
					}
				}
			}
		}

	}()
}
