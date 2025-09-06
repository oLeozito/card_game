.PHONY: servidor cliente

servidor:
	docker compose up servidor #docker compose up -d servidor

cliente:
	docker compose run --rm --service-ports cliente
