package handler

import "github.com/wagslane/go-rabbitmq"

type UserSettingHandlerI interface {
	UpdateByUserID() (*rabbitmq.Consumer, error)
}
