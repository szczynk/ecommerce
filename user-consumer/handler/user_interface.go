package handler

import "github.com/wagslane/go-rabbitmq"

type UserHandlerI interface {
	Create() (*rabbitmq.Consumer, error)
	UpdateByID() (*rabbitmq.Consumer, error)
}
