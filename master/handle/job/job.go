/**
*FileName: job
*Create on 2018-12-18 17:40
*Create by mok
 */

package job

import (
	"crontab/master/handle"
	"github.com/julienschmidt/httprouter"
	"net/http"
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

func GetJob(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	handle.SendResponse(w, &handle.Response{
		Code:    0,
		Message: "成功",
	})
}
