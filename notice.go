package main

const (
	noticeTitle      = "考试服务工具"
	noticeMsgSuccess = "运行考试服务成功"
	noticeMsgFailed  = "启动考试服务失败"
	noticeBtnOK      = "确定"
)

func showStartupNotice(success bool) {
	msg := noticeMsgSuccess
	if !success {
		msg = noticeMsgFailed
	}
	showNativeNotice(noticeTitle, msg, success)
}
