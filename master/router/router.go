/**
*FileName: main
*Create on 2018-12-18 17:05
*Create by mok
 */

package router

import (
	"crontab/master/handle/job"
	"github.com/julienschmidt/httprouter"
)

func LoadRouter(r *httprouter.Router) {
	r.POST("/job", job.CreateJob)
	r.GET("/job", job.GetJob)
}
