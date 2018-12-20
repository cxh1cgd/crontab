/**
*FileName: handle
*Create on 2018-12-18 18:53
*Create by mok
 */

package handle

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func SendResponse(w http.ResponseWriter, resp *Response) {
	data, _ := json.Marshal(resp)
	w.Write(data)
}
