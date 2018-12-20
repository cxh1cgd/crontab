/**
*FileName: job
*Create on 2018-12-18 17:28
*Create by mok
 */

package job

import (
	"context"
	"crontab/common/protocol"
	"encoding/json"
	"github.com/satori/go.uuid"
	"go.etcd.io/etcd/clientv3"
	"log"
	"sync"
	"time"
)

var Manager *jobManager

var p = sync.Pool{
	New: func() interface{} {
		return &jobs{l: new(sync.Mutex), m: make(map[string]*Job, 1000)}
	},
}

func init() {
	if err := InitJobManager(); err != nil {
		log.Fatalf("初始化任务管理失败，程序即将关闭，原因:%s", err)
	}
	Manager.asyncSave() //异步保存任务到etcd中
}

type jobManager struct {
	//ch chan <- protocol.Job    //冗余队列
	client *clientv3.Client //etcd client

	*jobs
	FailedJobs *jobs //彻底失败的任务列表
	//Done chan struct{}      //用于关闭任务管理操作
	OnSend chan struct{}
}

type jobs struct {
	l *sync.Mutex
	m map[string]*Job
}

type Job struct {
	j     *protocol.Job
	retry int //添加到etcd中失败后，重新添加次数
}

//初始化任务管理
func InitJobManager() error {
	Manager = &jobManager{
		client: nil,
		jobs:   p.Get().(*jobs),
		FailedJobs: &jobs{
			m: make(map[string]*Job),
		},
	}
	return Manager.connetctToEtcd()
}

//连接etcd
func (manager *jobManager) connetctToEtcd() error {
	c := clientv3.Config{
		Endpoints:          []string{},
		DialTimeout:        10 * time.Second,
		MaxCallSendMsgSize: 10 * 1024 * 1024,
	}
	client, err := clientv3.New(c)
	if err == nil {
		manager.client = client
	}
	return err
}

//创建任务
func (jm *jobManager) AddJob(name, command, exp string) {
	id, _ := uuid.NewV4()
	jobId := id.String()
	job := &Job{
		j: &protocol.Job{
			JobId: jobId, Command: command, Expression: exp,
		},
		retry: 5,
	}
	jm.jobs.l.Lock()
	jm.jobs.m[jobId] = job
	jm.jobs.l.Unlock()
}

//异步保存jobs到etcd上
func (manager *jobManager) asyncSave() {
	go func() {
		if manager.client == nil {
			log.Fatal("etcd连接为空，程序即将关闭")
		}
		if len(manager.m) >= 800 {
			manager.OnSend <- struct{}{}
		}
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				manager.save()
			case <-manager.OnSend:
				manager.save()
			}
		}
	}()
}

func (manager *jobManager) save() {
	for i := 0; i < 10; i++ {
		go func() {
			for k, v := range manager.jobs.m {

				data, _ := json.Marshal(v)
				_, err := manager.client.Put(context.TODO(), k, string(data))
				if err != nil {
					log.Printf("添加任务失败，稍后将自动重新添加,k:%s v:%v\n", k, *v)
					if v.retry == 0 {
						log.Printf("该任务无法添加，,k:%s v:%v\n", k, *v)
						manager.FailedJobs.m[k] = v
					}
					v.retry--
					continue
				}
				delete(manager.jobs.m, k)
			}
		}()
	}
}
