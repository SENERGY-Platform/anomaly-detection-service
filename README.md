Anomaly-Detection-Service
===========

## Documentation

Hand-written notes on this service live in [docs/](docs/):

- [Aspects arrive as a list](docs/aspects-arrive-as-a-list.md) — registering a handler for one
  or several aspects, and what the deprecated single aspect id still guarantees.

## MongoDB configuration

Set in `config.json`, overridden by environment variables:

| Variable                   | Default                     | Meaning                                                   |
|----------------------------|-----------------------------|-----------------------------------------------------------|
| `MONGO_URL`                | `mongodb://localhost:27017` | connection string including the scheme                    |
| `MONGO_USER`               | empty                       | user name; empty means no authentication                  |
| `MONGO_PASSWORD`           | empty                       | password; required when `MONGO_USER` is set, never logged |
| `MONGO_AUTH_SOURCE`        | `admin`                     | database the user is defined in                           |
| `MONGO_DATABASE`           | `anomaly_detection`         | database of this service; must not be empty               |
| `MONGO_ANOMALY_COLLECTION` | `anomalies`                 | collection for detected anomalies                         |

Keep credentials out of `MONGO_URL` and use `MONGO_USER`/`MONGO_PASSWORD`: the URL is not masked.
When `MONGO_USER` is set, `MONGO_USER`, `MONGO_PASSWORD` and `MONGO_AUTH_SOURCE` (default `admin`)
replace the user, password, `authSource` and `authMechanism` given in `MONGO_URL`; the mechanism is then
negotiated with the server. At startup the service lists the collections of `MONGO_DATABASE` and exits
if MongoDB is unreachable or the credentials are wrong, missing or lack rights on that database.

## Tests

CI runs `go test -p 1 -short ./...`. Without `-short`, `TestNewAuthenticatedStartupCheck` also runs
against a throwaway MongoDB with access control when these are set; it creates and drops its own users
and databases:

| Variable                   | Meaning                                        |
|----------------------------|------------------------------------------------|
| `MONGO_AUTH_TEST_URL`      | connection string without credentials          |
| `MONGO_AUTH_TEST_USER`     | root user of that server, defined in `admin`   |
| `MONGO_AUTH_TEST_PASSWORD` | password of the root user                      |

