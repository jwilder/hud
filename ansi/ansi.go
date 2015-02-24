package ansi

const (
	ColorRed         = "0;31"
	ColorGreen       = "0;32"
	ColorOrange      = "0;33"
	ColorBlue        = "0;34"
	ColorPurple      = "0;35"
	ColorCyan        = "0;36"
	ColorLightGray   = "0;37"
	ColorDarkGay     = "1;30"
	ColorLightRed    = "1;31"
	ColorLightGreen  = "1;32"
	ColorYellow      = "1;33"
	ColorLightBlue   = "1;34"
	ColorLightPurple = "1;35"
	ColorLightCyan   = "1;36"
	ColorWhite       = "1;37"
)

func StripAnsiControl(buf []byte) []byte {
	ret := make([]byte, len(buf))
	i := 0
	j := 0
	for {

		if i >= len(buf) {
			break
		}

		if buf[i] == 27 && i+1 < len(buf) && buf[i+1] == '[' {
			start := j
			done := false
			for {

				if i >= len(buf) || done {
					break
				}

				c := buf[i]
				ret[j] = c
				j += 1
				i += 1
				switch c {
				case 'A', 'B', 'C', 'D', 'H', 'f', 's', 'u', 'J', 'K', 'p':
					j = start
					done = true
					break
				case 'm', 'h', 'l':
					done = true
					break
				}
			}
		} else {
			c := buf[i]
			ret[j] = c
			j += 1
			i += 1
		}
	}
	return ret[:j]
}

func StripAnsi(buf []byte) []byte {
	ret := make([]byte, len(buf))
	i := 0
	j := 0
	for {

		if i >= len(buf) {
			break
		}

		if buf[i] == 27 && i+1 < len(buf) && buf[i+1] == '[' {
			start := j
			done := false
			for {

				if i >= len(buf) || done {
					break
				}

				c := buf[i]
				ret[j] = c
				j += 1
				i += 1
				switch c {
				case 'A', 'B', 'C', 'D', 'H', 'f', 's', 'u', 'J', 'K', 'p', 'm', 'h', 'l':
					j = start
					done = true
					break
				}
			}
		} else {
			c := buf[i]
			ret[j] = c
			j += 1
			i += 1
		}
	}
	return ret[:j]
}
