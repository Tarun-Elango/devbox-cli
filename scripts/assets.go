package scripts

import _ "embed"

//go:embed check.bash
var CheckBash []byte

//go:embed devbox-idle-stop.service
var IdleStopService []byte

//go:embed devbox-idle-stop.timer
var IdleStopTimer []byte

//go:embed devbox-idle-stop-boot.service
var IdleStopBootService []byte
