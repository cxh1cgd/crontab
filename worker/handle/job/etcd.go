/**
*FileName: job
*Create on 2018-12-28 14:39
*Create by mok
 */

package job

import (
	"context"
	"crontab/worker/config"
	"errors"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"log"
	"time"
)

var (
	Client *clientv3.Client
	Lease  clientv3.Lease
	Kv     clientv3.KV
)

func InitEtcd() {
	c := clientv3.Config{
		Endpoints:          config.Conf.Endpoints,
		MaxCallSendMsgSize: 10 * 1024 * 1024,
		DialTimeout:        5 * time.Second,
	}
	var err error
	fmt.Println(config.Conf.Endpoints)
	Client, err = clientv3.New(c)
	if err != nil {
		log.Fatalf("无法连接到etcd,启动进程失败,原因:%s", err.Error())
	}
	Lease = clientv3.NewLease(Client)
	Kv = clientv3.NewKV(Client)
}

//分布式锁
type EtcdLock struct {
	Key     string
	leaseId clientv3.LeaseID
	txn     clientv3.Txn
	cancel  context.CancelFunc
	Ttl     int64
}

func (l *EtcdLock) init() error {
	l.txn = Kv.Txn(context.TODO())
	leaseResp, err := Lease.Grant(context.TODO(), l.Ttl)
	if err != nil {
		return err
	}
	var ctx, cancel = context.WithCancel(context.TODO())
	l.cancel = cancel
	l.leaseId = leaseResp.ID
	_, err = Lease.KeepAlive(ctx, l.leaseId)
	return err
}

func (l *EtcdLock) Lock() error {
	err := l.init()
	if err != nil {
		return err
	}
	l.txn.If(clientv3.Compare(clientv3.CreateRevision(l.Key), "=", 0)).
		Then(clientv3.OpPut(l.Key, "", clientv3.WithLease(l.leaseId))).
		Else()
	resp, err := l.txn.Commit()
	if err == nil {
		if !resp.Succeeded { //判断txn.if条件是否成立
			return errors.New("获取锁失败")
		}
	}
	return err
}

func (l *EtcdLock) Unlock() {
	l.cancel()
	Lease.Revoke(context.TODO(), l.leaseId)
}
