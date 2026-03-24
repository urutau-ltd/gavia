package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

type BaseData struct {
	Title      string
	FooterData FooterData
}

type FooterData struct {
	RenderTime  string
	AileVersion string
	GoVersion   string
}

func NewBaseData(title string, start time.Time) BaseData {
	return BaseData{
		Title: title,
		FooterData: FooterData{
			RenderTime: fmt.Sprintf(
				"%.2fs",
				time.Since(start).Seconds(),
			),
			AileVersion: "v1.1.0",
			GoVersion: strings.Trim(
				runtime.Version(),
				"go",
			),
		},
	}
}
