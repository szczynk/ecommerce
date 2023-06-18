package service

import "user-consumer-go/model"

type UserServiceI interface {
	Create(user *model.User) error
	UpdateByID(user *model.User) error
}
