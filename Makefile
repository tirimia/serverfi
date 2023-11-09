# Please stop using Make as a task runner, you're not a C developer, you're writing a go/node/whatever microservice and are rightfully avoiding a bash switch case
.DEFAULT_GOAL := just
.PHONY: just just-exists

just-exists: ; @which just > /dev/null || echo "Just is not installed. Please MAKE sure it is. See https://github.com/casey/just#installation" && exit 1

just: just-exists
	@echo "Just run just..."
	@just -l
	@echo "P.S. : if you're a Makefile fanatic, you can just pass the commands through make"

.DEFAULT: just-exists
	@just $@
