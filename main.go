package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// to be changed to env var for "prod"
const dbURL           = "postgres://postgres:admin@localhost:5432/postgres"
var sqlErrorMessage   = gin.H{"message": "Unknown SQL error. Contact Admins. Or don't."}
var sqlError 		  = errors.New("SQL Error")
var MissingTokenError = errors.New("no token supplied")
var ExpiredTokenError = errors.New("token expired")
var InvalidTokenError = errors.New("token invalid")

// https://github.com/gin-gonic/gin/issues/932#issuecomment-306242400

type Env struct {
	db *pgxpool.Pool
}

type user struct {
	Name       string    `json:"name"`
	Token      uuid.UUID `json:"token"`
	LastAccess time.Time `json:"lastAccess"`
}

func newUser(username string) user {
	return user{
		Name:       username,
		Token:      uuid.New(),
		LastAccess: time.Now()}
}

type match struct {
	ID    uuid.UUID
	Host  *user
	Guest *user
}

// TODO: don't return errors from routing functions. they are the baseline
func (e *Env) RemoveUser(username string, c *gin.Context) {
	tx, err := e.db.Begin(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), "DELETE FROM tokens WHERE username = $1", username)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	_, err = tx.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	return
}

func (e *Env) CheckExpiryAndDelete(token uuid.UUID, c *gin.Context) (bool, error) {
	rows, _ := e.db.Query(context.Background(), "SELECT COUNT(*) FROM tokens WHERE token = $1", token.String())
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return false, err
	}

	if matches == 0 {
		c.IndentedJSON(http.StatusForbidden, gin.H{"message": "token doesn't exist"})
		return true, nil
	}

	rows, err = e.db.Query(context.Background(), "DELETE FROM tokens WHERE token = $1 AND NOW() - lastaccess > INTERVAL '10 minutes' RETURNING username", token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return false, err
	}

	username, err := pgx.CollectOneRow(rows, pgx.RowTo[string])
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return false, nil
		default:
			c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
			return false, err
		}
	}

	e.RemoveUser(username, c)
	return true, nil
}

func (e *Env) postUsers(c *gin.Context) {
	username, exists := c.GetPostForm("username") // TODO: check username length

	if username == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no username supplied"})
		return
	}

	rows, _ := e.db.Query(context.Background(), "SELECT COUNT(*) FROM users WHERE username = $1", username)
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if matches > 0 {
		c.IndentedJSON(http.StatusConflict, gin.H{"message": "username taken"})
		return
	}

	tx, err := e.db.Begin(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	defer tx.Rollback(context.Background())

	newu := newUser(username)
	_, err = tx.Exec(context.Background(), "INSERT INTO users (username) VALUES ($1)", newu.Name)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	_, err = tx.Exec(context.Background(), "INSERT INTO tokens (username, token, lastaccess) VALUES ($1, $2, $3)", newu.Name, newu.Token, newu.LastAccess)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	_, err = tx.Exec(context.Background(), "INSERT INTO user_status (username, user_status) VALUES ($1, $2)", newu.Name, "idle")
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{"token": newu.Token, "message": "user created"})
}

func (e *Env) extendSession(c *gin.Context) error {
	token, exists := c.GetPostForm("token")

	if token == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no token supplied"})
		return MissingTokenError
	}

	tokenUUID, err := uuid.Parse(token)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "token not UUID"})
		return err
	}

	expired, err := e.CheckExpiryAndDelete(tokenUUID, c)
	if err != nil {
		return err
	}

	if expired {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
		return ExpiredTokenError
	}

	_, err = e.db.Exec(context.Background(), "UPDATE tokens SET lastaccess = NOW() WHERE token = $1", token)
	return err
}

func (e *Env) extendSessionRequest(c *gin.Context) {
	if err := e.extendSession(c); err != nil {
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "session extended"})
}

func (e *Env) joinLobby(c *gin.Context) {
	if err := e.extendSession(c); err != nil {
		return
	}

	rows, _ := e.db.Query(context.Background(), "SELECT COUNT(*) FROM user_status WHERE user_status = $1", "hosting")
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if matches < 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no match hosts found, consider hosting"})
		return
	}

	// TODO:
	// --get number of users in hosting status
	// --if none, return... resource unavailable?
	// if some, get one random host, change status of both users to playing, put them in match
	// match should be in memory, use a thread safe map
	// I guess get/match should return a redirect if on the wrong server, match table should store IP
}

func (e *Env) hostMatch(c *gin.Context) {
	if err := e.extendSession(c); err != nil {
		return
	}

	if err != nil {
		return
	}

	if lobby.Contains(user) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "user already trying to host match"})
		return
	}

	lobby.Add(user)
	c.IndentedJSON(http.StatusOK, gin.H{"message": "user now looking for other players"})
}

func (e *Env) unhostMatch(c *gin.Context) {
	if err := e.extendSession(c); err != nil {
		return
	}

	if !lobby.Contains(user) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "user not trying to host match"})
	}

	e.RemoveUser(user)
	c.IndentedJSON(http.StatusOK, gin.H{"message": "user no longer hosting"})
}

func main() {
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Printf("Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	// FIXME: but don't fix yet. make schema first
	// TODO: do user side first, then schema for matches, then matches logic changes, then schema for games, etc etc
	router := gin.Default()
	env := &Env{db: dbpool}
	router.POST("/user/:username", env.postUsers) // FIXME: router functions can't return errors
	router.POST("/extendSession/:token", env.extendSessionRequest)
	router.POST("/joinLobby/:token", env.joinLobby)
	router.POST("/hostMatch/:token", hostMatch)
	router.DELETE("/hostmatch/:token", unhostMatch)

	router.Run("localhost:8080")
}
