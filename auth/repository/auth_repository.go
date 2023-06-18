package repository

import (
	"auth-go/helper/timeout"
	"auth-go/model"
	"auth-go/package/rmq"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/redis/go-redis/v9"
	"github.com/wagslane/go-rabbitmq"
)

type AuthRepository struct {
	db      *sql.DB
	redis   *redis.Client
	rmqConn *rabbitmq.Conn
}

func NewAuthRepository(db *sql.DB, redis *redis.Client, rmqConn *rabbitmq.Conn) AuthRepositoryI {
	repo := new(AuthRepository)
	repo.db = db
	repo.redis = redis
	repo.rmqConn = rmqConn
	return repo
}

func (repo *AuthRepository) isEmailUnique(ctx context.Context, user *model.User) error {
	sqlQuery := `
	SELECT COUNT(*) FROM users WHERE email = $1
	`
	stmt, errStmt := repo.db.PrepareContext(ctx, sqlQuery)
	if errStmt != nil {
		return errStmt
	}
	defer stmt.Close()

	count := 0
	errScan := stmt.QueryRowContext(ctx, user.Email).Scan(&count)
	if errScan != nil {
		return errScan
	}

	if count > 0 {
		return errors.New("duplicated key not allowed")
	}
	return nil
}

func (repo *AuthRepository) Create(user *model.User) error {
	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	// check if an email is unique before inserting

	if errEmailUnique := repo.isEmailUnique(ctx, user); errEmailUnique != nil {
		return errEmailUnique
	}

	userBytes, _ := json.Marshal(user)
	// if errJSON != nil {
	// 	return errJSON
	// }

	errPub := rmq.PublishWithContext(
		ctx, repo.rmqConn,
		[]string{"user.created"}, "application/json",
		true, userBytes,
		"", "",
		"user", "topic",
	)
	if errPub != nil {
		return errPub
	}
	errPub2 := rmq.PublishWithContext(
		ctx, repo.rmqConn,
		[]string{"my_routing_key"}, "application/json",
		true, userBytes,
		"", "",
		"events", "direct",
	)
	if errPub2 != nil {
		return errPub2
	}

	return nil
}

func (repo *AuthRepository) FirstOrCreate(user *model.User) (*model.User, error) {
	foundUser, err := repo.LoginByEmail(user.Email)
	if err == nil {
		return foundUser, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if errCreate := repo.Create(user); errCreate != nil {
		return nil, errCreate
	}

	// Retry getting the user after creation
	foundUser, err = repo.retryLoginByEmail(user.Email, 3, 3*time.Second)
	if err != nil {
		return nil, err
	}

	return foundUser, nil
}

// func (repo *AuthRepository) retryLoginByEmail(email string, delay, timeout time.Duration) (*model.User, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()

// 	respChan := make(chan *struct {
// 		user *model.User
// 		err  error
// 	}, 1)

// 	go func() {
// 		time.Sleep(delay)
// 		user, err := repo.LoginByEmail(email)
// 		respChan <- &struct {
// 			user *model.User
// 			err  error
// 		}{user: user, err: err}
// 	}()

// 	select {
// 	case res := <-respChan:
// 		return res.user, res.err
// 	case <-ctx.Done():
// 		return nil, fmt.Errorf("timeout")
// 	}
// }

func (repo *AuthRepository) retryLoginByEmail(email string, attempts uint, delay time.Duration) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user *model.User
	var err error

	err = retry.Do(
		func() error {
			user, err = repo.LoginByEmail(email)
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

func (repo *AuthRepository) LoginByEmail(email string) (*model.User, error) {
	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	sqlQuery := `
	SELECT id, username, email, password, role, provider,
	    	  full_name, age, image_url, created_at, updated_at
	FROM users 
	WHERE email = $1
	LIMIT 1
	`
	stmt, errStmt := repo.db.PrepareContext(ctx, sqlQuery)
	if errStmt != nil {
		return nil, errStmt
	}
	defer stmt.Close()

	user := new(model.User)
	row := stmt.QueryRowContext(ctx, email)
	scanErr := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Provider,
		&user.FullName, &user.Age, &user.ImageURL,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if scanErr != nil {
		return nil, scanErr
	}

	return user, nil
}

func (repo *AuthRepository) SetRefreshToken(token string, dataByte []byte, refreshTokenDur time.Duration) error {
	return repo.redis.Set(
		context.Background(),
		"refresh_token:"+token, dataByte, refreshTokenDur,
	).Err()
}

func (repo *AuthRepository) GetByRefreshToken(token string) (*model.RefreshToken, error) {
	refreshToken := new(model.RefreshToken)

	cachedData, errGetCache := repo.redis.Get(
		context.Background(), "refresh_token:"+token,
	).Bytes()
	if errGetCache != nil {
		return nil, errGetCache
	}

	errJSONUn := json.Unmarshal(cachedData, &refreshToken)
	if errJSONUn != nil {
		return nil, errJSONUn
	}

	return refreshToken, nil
}
