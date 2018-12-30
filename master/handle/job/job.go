/**
*FileName: job
*Create on 2018-12-18 17:40
*Create by mok
 */

package job

import (
	"crontab/master/handle"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strings"
)

//添加任务
func CreateJob(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	vals := r.PostForm
	name := vals["name"][0]
	command := vals["command"][0]
	express := vals["expression"][0]
	if command == "" || name == "" || express == "" {
		handle.SendResponse(w, &handle.Response{
			Code:    400,
			Message: "非法参数",
		})
		return
	}
	Manager.AddJob(name, command, express)
	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "添加任务成功",
	})
}

//获取单个任务
func GetJob(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	jobID := strings.Trim(p.ByName("jobId"), " ")
	if jobID == "" {
		handle.SendResponse(w, &handle.Response{
			Code:    400,
			Message: "非法参数",
		})
		return
	}
	job, err := Manager.GetJob(jobID)
	if err != nil {
		handle.SendResponse(w, &handle.Response{
			Code:    500,
			Message: "获取任务失败",
		})
		return
	}
	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "成功",
		Data:    map[string]interface{}{"job": job},
	})
}

//获取所有任务
func GetJobs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	jobs, err := Manager.GetJobs()
	if err != nil {
		handle.SendResponse(w, &handle.Response{
			Code:    500,
			Message: "获取任务列表失败",
		})
		return
	}
	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "成功",
		Data:    map[string]interface{}{"jobs": jobs},
	})
}

//删除任务
func DeleteJob(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	jobID := p.ByName("jobId")
	if jobID == "" {
		handle.SendResponse(w, &handle.Response{
			Code:    400,
			Message: "非法参数",
		})
		return
	}
	if err := Manager.DeleteJob(jobID); err != nil {
		handle.SendResponse(w, &handle.Response{
			Code:    500,
			Message: "删除任务失败",
		})
		log.Print(err.Error())
		return
	}

	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "成功",
	})
}

//更新任务
func Update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	r.ParseForm()
	vals := r.PostForm
	jobId := vals["jobId"][0]
	name := vals["name"][0]
	command := vals["command"][0]
	express := vals["expression"][0]
	if command == "" || name == "" || express == "" {
		handle.SendResponse(w, &handle.Response{
			Code:    400,
			Message: "非法参数",
		})
		return
	}
	err := Manager.UpdateJob(jobId, name, command, express)
	if err != nil {
		handle.SendResponse(w, &handle.Response{
			Code:    500,
			Message: "更新任务失败",
		})
		return
	}
	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "OK",
	})
	log.Printf("更新任务成功，任务ID：%s\n", jobId)
}

/*//获取失败任务列表
func GetFailedJobs(w http.ResponseWriter,r *http.Request,_ httprouter.Params){
	handle.SendResponse(w,&handle.Response{
		Code:0,
		Message:"成功",
	})
}*/
