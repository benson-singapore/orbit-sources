package main

import "fmt"

var errEmptyArticleID = fmt.Errorf("empty article id")

type httpStatusError int

func (e httpStatusError) Error() string {
	return fmt.Sprintf("http status %d", int(e))
}

func errHTTPStatus(code int) error {
	return httpStatusError(code)
}
