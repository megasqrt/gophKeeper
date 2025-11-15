# GophKeeper

Менеджер паролей GophKeeper

## Структура проекта

```
/home/kan/src/gophKeeper/
├── client
│   ├── cmd
│   │   └── main.go
│   ├── go.mod
│   └── internal
│       ├── app
│       │   └── app.go
│       ├── config
│       │   └── config.go
│       ├── delivery
│       │   └── cli
│       │       └── cli.go
│       ├── services
│       │   └── sync.go
│       └── transport
│           └── grpc
│               └── client.go
├── proto
│   └── gophkeeper.proto
├── server
│   ├── cmd
│   │   └── main.go
│   ├── go.mod
│   └── internal
│       ├── app
│       │   └── app.go
│       ├── config
│       │   └── config.go
│       ├── delivery
│       │   └── grpc
│       │       └── server.go
│       ├── domain
│       │   ├── model
│       │   │   └── user.go
│       │   └── repository
│       │       └── user_repository.go
│       └── services
│           └── auth.go
├── tz.md
└── .gitignore
```
