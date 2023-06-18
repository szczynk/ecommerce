package repository

import (
	"api-gateway-go/helper/timeout"
	"api-gateway-go/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type ShortenRepo struct {
	db    *sql.DB
	redis *redis.Client
}

func NewShortenRepo(db *sql.DB, redis *redis.Client) ShortenRepoI {
	repo := new(ShortenRepo)
	repo.db = db
	repo.redis = redis
	return repo
}

func (repo *ShortenRepo) Get(hashedURL string) (*model.APIManagement, error) {
	apiManagement := new(model.APIManagement)

	// note: implemetation of cache aside of read cache strategy
	cachedData, errGetCache := repo.redis.Get(
		context.Background(), "hashed_path:"+hashedURL,
	).Bytes()
	if errGetCache == nil {
		errJSONUn := json.Unmarshal(cachedData, &apiManagement)
		if errJSONUn != nil {
			return nil, errJSONUn
		}
		return apiManagement, nil
	}

	if !errors.Is(errGetCache, redis.Nil) {
		return nil, errGetCache
	}

	var errGetDB error
	apiManagement, errGetDB = repo.getHashURLFromDatabase(hashedURL)
	if errGetDB != nil {
		return nil, errGetDB
	}

	dataByte, _ := json.Marshal(apiManagement)
	// if errJSON != nil {
	// 	return nil, errJSON
	// }

	errSetCache := repo.redis.Set(
		context.Background(),
		"hashed_path:"+hashedURL, dataByte, 10*time.Minute,
	).Err()
	if errSetCache != nil {
		return nil, errSetCache
	}

	return apiManagement, nil
}

func (repo *ShortenRepo) getHashURLFromDatabase(hashedURL string) (*model.APIManagement, error) {
	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	sqlQuery := `
	SELECT id, api_name, service_name, endpoint_url, 
				 hashed_endpoint_url, is_available, need_bypass,
				 created_at, updated_at 
	FROM api_managements 
	WHERE hashed_endpoint_url = $1 AND is_available = TRUE 
	LIMIT 1
	`
	stmt, errStmt := repo.db.PrepareContext(ctx, sqlQuery)
	if errStmt != nil {
		return nil, errStmt
	}
	defer stmt.Close()

	apiManagement := new(model.APIManagement)
	row := stmt.QueryRowContext(ctx, hashedURL)
	scanErr := row.Scan(
		&apiManagement.ID, &apiManagement.APIName, &apiManagement.ServiceName, &apiManagement.EndpointURL,
		&apiManagement.HashedEndpointURL, &apiManagement.IsAvailable, &apiManagement.NeedBypass,
		&apiManagement.CreatedAt, &apiManagement.UpdatedAt,
	)
	if scanErr != nil {
		return nil, scanErr
	}

	return apiManagement, nil
}

func (repo *ShortenRepo) Create(apiManagement *model.APIManagement) (*model.APIManagement, error) {
	ctx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	sqlQuery := `
	INSERT INTO api_managements (api_name, service_name, endpoint_url, hashed_endpoint_url, is_available, need_bypass)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, updated_at
	`
	stmt, errStmt := repo.db.PrepareContext(ctx, sqlQuery)
	if errStmt != nil {
		return nil, errStmt
	}
	defer stmt.Close()

	scanErr := stmt.QueryRowContext(
		ctx,
		&apiManagement.APIName, &apiManagement.ServiceName,
		&apiManagement.EndpointURL, &apiManagement.HashedEndpointURL,
		&apiManagement.IsAvailable, &apiManagement.NeedBypass,
	).Scan(
		&apiManagement.ID, &apiManagement.CreatedAt, &apiManagement.UpdatedAt,
	)
	if scanErr != nil {
		return nil, scanErr
	}

	return apiManagement, nil
}
