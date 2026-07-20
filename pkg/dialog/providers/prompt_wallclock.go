package providers

import (
	"fmt"
	"time"
)

// WallClockPromptHint injects authoritative local date/time so the model
// can resolve 「今天/明天/下午」 without hallucinating years.
func WallClockPromptHint(now time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	now = now.In(loc)
	days := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	tomorrow := now.AddDate(0, 0, 1)
	return fmt.Sprintf(
		"【当前时间】时区 Asia/Shanghai：今天=%s（%s），明天=%s，此刻=%s。"+
			"用户说「今天/明天/后天/本周/下午X点」时必须据此推算 scheduledDate(YYYY-MM-DD) 与 startTime/endTime(HH:mm)；"+
			"禁止编造或沿用训练数据里的旧年份。预约排课也可先调 current_time 复核。",
		now.Format("2006-01-02"),
		days[int(now.Weekday())],
		tomorrow.Format("2006-01-02"),
		now.Format("15:04"),
	)
}
