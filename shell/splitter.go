package shell

// func splitter(line string) (r []string) {
// 	var buff bytes.Buffer

// 	s := false
// 	quotecounter := 0
// 	for _, val := range line {
// 		if val == '"' || val == '\'' || val == '`' {
// 			switch quotecounter {
// 			case 0:
// 				s = !s
// 				quotecounter = 1
// 			case 1:
// 				break
// 			}
// 		}
// 		if (val == ' ' || val == ':') && !s {
// 			r = append(r, buff.String())
// 			buff.Reset()
// 			continue
// 		}
// 		buff.WriteRune(val)
// 	}

// 	if buff.Available() > 0 {
// 		r = append(r, buff.String())
// 	}

// 	return r
// }
