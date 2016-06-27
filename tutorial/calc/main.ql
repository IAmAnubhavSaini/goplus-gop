include "calc.ql"

main { // 使用main关键字将主程序括起来，是为了避免其中用的局部变量比如 err 对其他函数造成影响

	calc = new Calculator
	engine, err = interpreter(calc, nil)
	if err != nil {
		fprintln(os.stderr, err)
		return 1
	}

	scanner = bufio.scanner(os.stdin)
	for scanner.scan() {
		line = strings.trim(scanner.text(), " \t\r\n")
		if line != "" {
			err = engine.eval(line)
			if err != nil {
				fprintln(os.stderr, err)
			} else {
				printf("> %v\n\n", calc.ret())
			}
		}
	}
}
