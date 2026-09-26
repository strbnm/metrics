# internal/repository

Этот пакет содержит реализацию работы с базой данных, а также со внешними сервисами.

Важно, чтобы репозиторий не содержал бизнес-логику.

Репозиторий реализует паттерн Repository и служит абстракцией над различными источниками данных, такими как:
- базы данных (PostgreSQL, MySQL и др.)
- внешние API
- файловые системы
- кэши (Redis, Memcached)
- другие источники данных.


### Запуск интеграционных тестов БД

```shell
# поднять тестовый PostgreSQL
docker run -d --name test_pg \
  -e POSTGRES_DB=test_metrics \
  -e POSTGRES_PASSWORD=postgres \
  -p 5433:5432 \
  postgres:16

# запустить контракт-тесты с тегом integration
go test -tags=integration -v ./internal/repository/...
```