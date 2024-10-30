package runtime

// HandleCrash 允许外部传入额外的异常处理
func HandleCrash(reallyCrash bool, handlers ...func(any)) {
	r := recover()
	for _, fn := range handlers {
		fn(r)
	}
	if r != nil && reallyCrash {
		panic(r)
	}
}
