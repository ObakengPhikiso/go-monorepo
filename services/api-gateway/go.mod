module github.com/ObakengPhikiso/monorepo/services/api-gateway

go 1.24

require github.com/ObakengPhikiso/monorepo/libs/shared v0.0.0-00010101000000-000000000000

require github.com/golang-jwt/jwt/v5 v5.2.2 // indirect

replace github.com/ObakengPhikiso/monorepo/libs/shared => ../../libs/shared
