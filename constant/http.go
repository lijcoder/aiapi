package constant

type HttpGeneralResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type HttpCustomError struct {
	Msg string `json:"msg"`
	Err error  `json:"_"`
}

func BuildHttpResponseSuccess(data any) HttpGeneralResp {
	return HttpGeneralResp{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
}

func BuildHttpResponseFail(msg string) HttpGeneralResp {
	return HttpGeneralResp{
		Code: -1,
		Msg:  msg,
	}
}
