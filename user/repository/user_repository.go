package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"
	"user-go/helper/timeout"
	"user-go/model"
	"user-go/package/rmq"

	"github.com/avast/retry-go/v4"
	"github.com/redis/go-redis/v9"
	"github.com/wagslane/go-rabbitmq"
)

type UserRepository struct {
	db      *sql.DB
	redis   *redis.Client
	rmqConn *rabbitmq.Conn
}

func NewUserRepository(db *sql.DB, redis *redis.Client, rmqConn *rabbitmq.Conn) UserRepositoryI {
	repo := new(UserRepository)
	repo.db = db
	repo.redis = redis
	repo.rmqConn = rmqConn
	return repo
}

func (repo *UserRepository) GetByID(userID uint) (*model.User, error) {
	user := new(model.User)

	cachedData, errGetCache := repo.redis.Get(
		context.Background(),
		"user_id:"+strconv.FormatUint(uint64(userID), 10),
	).Bytes()
	if errGetCache == nil {
		errJSONUn := json.Unmarshal(cachedData, &user)
		if errJSONUn != nil {
			return nil, errJSONUn
		}
		return user, nil
	}

	if !errors.Is(errGetCache, redis.Nil) {
		return nil, errGetCache
	}

	var errGetDB error
	user, errGetDB = repo.getByUserIDFromDatabase(userID)
	if errGetDB != nil {
		return nil, errGetDB
	}

	dataByte, _ := json.Marshal(user)
	// if errJSON != nil {
	// 	return nil, errJSON
	// }

	errSetCache := repo.redis.Set(
		context.Background(),
		"user_id:"+strconv.FormatUint(uint64(userID), 10), dataByte, 10*time.Minute,
	).Err()
	if errSetCache != nil {
		return nil, errSetCache
	}

	return user, nil
}

func (repo *UserRepository) getByUserIDFromDatabase(userID uint) (*model.User, error) {
	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	sqlQuery := `
	SELECT users.id, users.username, users.email, users.role, users.provider,
				 users.full_name, users.age, users.image_url, users.created_at, users.updated_at, 
				 user_settings.notification, user_settings.dark_mode, 
				 languages.name AS language
	FROM users 
	INNER JOIN user_settings on users.id = user_settings.user_id
	INNER JOIN languages on user_settings.language_id= languages.id
	WHERE users.id = $1
	LIMIT 1
	`
	stmt, errStmt := repo.db.PrepareContext(ctx, sqlQuery)
	if errStmt != nil {
		return nil, errStmt
	}
	defer stmt.Close()

	user := new(model.User)
	row := stmt.QueryRowContext(ctx, userID)
	scanErr := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Role, &user.Provider,
		&user.FullName, &user.Age, &user.ImageURL, &user.CreatedAt, &user.UpdatedAt,
		&user.UserSetting.Notification, &user.UserSetting.DarkMode,
		&user.UserSetting.Language.Name,
	)
	if scanErr != nil {
		return nil, scanErr
	}

	return user, nil
}

func (repo *UserRepository) UpdateByID(profile *model.User) (*model.User, error) {
	userID := profile.ID

	errDelCache := repo.redis.Del(
		context.Background(),
		"user_id:"+strconv.FormatUint(uint64(userID), 10),
	).Err()
	if errDelCache != nil {
		return nil, errDelCache
	}

	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	profileBytes, _ := json.Marshal(profile)
	// if errJSON != nil {
	// 	return nil, errJSON
	// }

	errPub := rmq.PublishWithContext(
		ctx, repo.rmqConn,
		[]string{"user.updated"}, "application/json",
		true, profileBytes,
		"", "",
		"user", "topic",
	)
	if errPub != nil {
		return nil, errPub
	}

	user, err := repo.retryGetByID(userID, 3, 3*time.Second)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *UserRepository) retryGetByID(userID uint, attempts uint, delay time.Duration) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user *model.User
	var err error

	err = retry.Do(
		func() error {
			user, err = repo.GetByID(userID)
			return err
		},
		retry.Attempts(attempts),
		retry.Delay(delay),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Attempt %d failed; retrying in %v", n, delay)
		}),
		retry.Context(ctx),
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
