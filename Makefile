.PHONY: env

env:
	@test -e .env || cp .env.example .env
