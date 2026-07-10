package scripts

import _ "embed"

//go:embed check.bash
var CheckBash []byte

//go:embed outpost-idle-stop.service
var IdleStopService []byte

//go:embed outpost-idle-stop.timer
var IdleStopTimer []byte

//go:embed outpost-idle-stop-boot.service
var IdleStopBootService []byte
