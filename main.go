package main

// int plugin_is_GPL_compatible;
import "C"

import (
	"log"

	"github.com/sigma/go-emacs"
)

func init() {
	emacs.Register(initModule)
}

func initModule(env emacs.Environment) {
	stdlib := env.StdLib()
	helloFunc := env.MakeFunction(hello, 1, "echo", nil)
	helloSym := stdlib.Intern("pyspa/echo")
	stdlib.Fset(helloSym, helloFunc)
	pyspaSym := stdlib.Intern("pyspa")
	stdlib.Provide(pyspaSym)
}

func hello(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	stdlib := ctx.Environment().StdLib()
	path, err := ctx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), err
	}
	log.Println(path)
	return stdlib.Nil(), nil
}

func main() {
}
