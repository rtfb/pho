
schema:
	sql-migrate up -config=./db/dbconfig.yaml

run:
	go run *.go
