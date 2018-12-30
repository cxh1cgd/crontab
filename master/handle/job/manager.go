/**
*FileName: job
*Create on 2018-12-18 17:28
*Create by mok
 */

package job

import (
	"context"
	"crontab/common/protocol"
	"crontab/master/config"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"go.etcd.io/etcd/clientv3"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var Manager *jobManager

const (
	JobPrefix string = "/jobs/exec/"
)

var p = sync.Pool{
	New: func() interface{} {
		return &protocol.Job{}
	},
}

func Init() {
	if err := initJobManager(); err != nil {
		log.Fatalf("初始化任务管理失败，程序即将关闭，原因:%s", err)
	}
	Manager.asyncSave() //异步保存任务到etcd中
	resp, err := Manager.client.Get(context.TODO(), "/test/", clientv3.WithPrefix())
	if err != nil {
		panic(err)
	}
	fmt.Println(string(resp.Kvs[0].Value))
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
	l sync.Mutex
	m map[string]*Job
}

type Job struct {
	J     *protocol.Job
	lock  uint32 //并发原子操作锁
	retry int    //添加到etcd中失败后，重新添加次数
}

//初始化任务管理
func initJobManager() error {
	Manager = &jobManager{
		client: nil,
		jobs: &jobs{
			m: make(map[string]*Job, 1000),
		},
		FailedJobs: &jobs{
			m: make(map[string]*Job),
		},
	}

	return Manager.connetctToEtcd()
}

//连接etcd
func (manager *jobManager) connetctToEtcd() error {
	c := clientv3.Config{
		Endpoints:          config.Conf.Endpoints,
		MaxCallSendMsgSize: 10 * 1024 * 1024,
		DialTimeout:        5 * time.Second,
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
	job := &Job{
		J: &protocol.Job{
			JobId: id.String(), Command: command, Expression: exp, Name: name, Times: 0,
		},
		retry: 5,
		lock:  0,
	}
	jobId := protocol.JobSaveDir + id.String()
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

		ticker := time.NewTicker(1 * time.Second)
		for {
			if len(manager.m) >= 800 {
				manager.OnSend <- struct{}{}
			}
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
				if atomic.CompareAndSwapUint32(&v.lock, 0, 1) {
					data, _ := json.Marshal(v.J)
					_, err := manager.client.Put(context.TODO(), k, string(data))
					if err != nil {
						if v.retry != 0 {
							v.retry--
							log.Printf("添加任务失败，剩余自动重新添加次数：%d,k:%s v:%v\n", v.retry, k, *v)
							continue
						}
						log.Printf("该任务无法添加,k:%s v:%v\n", k, *v)
						manager.FailedJobs.m[k] = v
					}
					log.Printf("任务添加成功,k:%s v:%v\n", k, *v)
					delete(manager.jobs.m, k)
				}
			}
		}()
	}
}

//获取一个任务
func (manager *jobManager) GetJob(key string) (*protocol.Job, error) {
	resp, err := manager.client.Get(context.TODO(), JobPrefix+key)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.New("任务不存在")
	}
	data := resp.Kvs[0].Value
	job := p.Get().(*protocol.Job)
	defer p.Put(job)
	err = json.Unmarshal(data, job)
	return job, err
}

//获取所有任务列表
func (manager *jobManager) GetJobs() ([]*protocol.Job, error) {
	resp, err := manager.client.Get(context.TODO(), protocol.JobSaveDir, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var jobs = make([]*protocol.Job, 0)
	for _, kv := range resp.Kvs {
		//var job = new(protocol.Job)
		job := p.Get().(*protocol.Job)
		if err = json.Unmarshal(kv.Value, job); err != nil {
			return nil, err
		}
		fmt.Println(job)
		jobs = append(jobs, job)
	}
	return jobs, nil
}

//删除任务
func (manager *jobManager) DeleteJob(key string) error {
	_, err := manager.client.Delete(context.TODO(), protocol.JobSaveDir+key)
	delete(manager.m, key)
	return err
}

func (manager *jobManager) UpdateJob(jobId string, name string, command string, expression string) error {
	resp, err := manager.client.Get(context.TODO(), protocol.JobSaveDir+jobId, clientv3.WithCountOnly())
	if err != nil {
		return err
	}
	if resp.Count == 0 {
		return errors.New("该任务不存在")
	}
	job := protocol.JobPool.Get().(*protocol.Job)
	job.JobId = jobId
	job.Name = name
	job.Command = command
	job.Expression = expression
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = manager.client.Put(context.TODO(), protocol.JobSaveDir+jobId, string(data))
	if err != nil {
		return err
	}
	return nil
}
