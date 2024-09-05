package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// to be changed for prod
var dbURL = "postgres://postgres:admin@localhost:5432/postgres"

// https://github.com/gin-gonic/gin/issues/932#issuecomment-306242400

type Env struct {
	db *pgxpool.Pool
}

type user struct {
	Name       string    `json:"name"`
	Token      uuid.UUID `json:"token"`
	LastAccess time.Time `json:"lastAccess"`
}

var sqlError = gin.H{"message": "Unknown SQL error. Contact Admins. Or don't."}

func (e *Env) RemoveUser(username string, c *gin.Context) error {
	tx, err := e.db.Begin(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlError)
		return err
	}

	defer tx.Rollback(context.Background())
	
	_, err = tx.Exec(context.Background(), "DELETE FROM tokens WHERE username = $1", username)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlError)
		return err
	}

	_, err = tx.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlError)
    	return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlError)
		return err
	}

	return nil
}

func (e *Env) CheckExpiryAndDelete(token uuid.UUID, c *gin.Context) (bool, error) {
	rows, err := e.db.Query(context.Background(), "DELETE FROM tokens WHERE token = $1 AND NOW() - lastaccess > INTERVAL '10 minutes' RETURNING username", token)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlError)
		return false, err
	}

	username, err := pgx.CollectOneRow(rows, pgx.RowTo[string])
	if err != nil { // works but maybe refactor
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return false, nil
		default:
			c.IndentedJSON(http.StatusInternalServerError, sqlError)
			return false, err
		}
	}

	return true, e.RemoveUser(username, c)
}

func newUser(username string) *user {
	user := user{
		Name:       username,
		Token:      uuid.New(),
		LastAccess: time.Now()}

	return &user
}

type match struct {
	ID    uuid.UUID
	Host  *user
	Guest *user
}

func postUsers(c *gin.Context) {
	username, exists := c.GetPostForm("username")

	if username == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no username supplied"})
		return
	}

	for _, u := range users {
		if u.Name == username {
			c.IndentedJSON(http.StatusConflict, gin.H{"message": "username taken"})
			return
		}
	}

	newu := newUser(username)
	users = append(users, *newu)
	c.IndentedJSON(http.StatusCreated, newu)
}

func extendSession(c *gin.Context) (*user, error) {
	token, exists := c.GetPostForm("token")

	if token == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no token supplied"})
		return nil, errors.New("no token supplied")
	}

	for i, u := range users {
		if u.Token.String() == token {
			if users.C(i) {
				c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
				return nil, errors.New("token expired")
			}

			users[i].LastAccess = time.Now()
			c.IndentedJSON(http.StatusOK, users[i])
			log.Println("User updated: ", users[i])
			return &users[i], nil
		}
	}

	c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "incorrect token"})
	return nil, errors.New("incorrect token")
}

func extendSessionRequest(c *gin.Context) {
	user, err := extendSession(c)

	if err != nil {
		return
	}
	c.IndentedJSON(http.StatusOK, user) //check if deref needed. also, all uses of user in response messages are probably borked and I might need to do a tostring function
}

func joinLobby(c *gin.Context) {
	user, err := extendSession(c)

	if err != nil {
		return
	}

	if lobby.Cardinality() == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no match hosts found"})
		return
	}

	//todo: nightmare logic
}

func hostMatch(c *gin.Context) {
	user, err := extendSession(c)

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
	user, err := extendSession(c)

	if err != nil {
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

	env := &Env{db: dbpool}

	// todo: needs fixing. but don't fix yet. make schema first
	// todo: do user side first, then schema for matches, then matches logic changes, then schema for games, etc etc
	router := gin.Default()
	router.POST("/newSession/:username", postUsers)
	router.POST("/extendSession/:token", extendSessionRequest)
	router.POST("/joinLobby/:token", joinLobby)
	router.POST("/hostMatch/:token", hostMatch)
	router.DELETE("/hostmatch/:token", unhostMatch)

	router.Run("localhost:8080")
}
