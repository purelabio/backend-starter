package main

/*
Unlike the logger methods `.Error` and `.Errorw`, this prints TWO stacktraces:
where the error was created (if possible), and where the error was logged.
`.Error` or `.Errorw` always puts the error stacktrace inside JSON, making it
hard to read in development.

Also unlike logger methods, this logs only non-nil errors.
*/
func logError(err error) {
	if err != nil {
		env.log.Errorf("%+v", err)
	}
}

func maybeLogError(ctx Ctx, err error) {
	if shouldLogError(ctx, err) {
		logError(err)
	}
}
