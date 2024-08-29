package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type user struct {
	Name       string    `json:"name"`
	Token      uuid.UUID `json:"token"`
	LastAccess time.Time `json:"lastAccess"`
}

type userlist []user

var users = userlist{}

func (u userlist) remove(s int) userlist {
	return append(u[:s], u[s+1:]...)
}

func (u userlist) checkExpiryAndDelete(i int) bool {
	if u[i].LastAccess.Add(time.Minute * 10).Before(time.Now()) {
		u.remove(i)
		return true
	}
	return false
}

func newUser(username string) *user {
	user := user{
		Name:       username,
		Token:      uuid.New(),
		LastAccess: time.Now()}

	return &user
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
			if users.checkExpiryAndDelete(i) {
				c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
				return nil, errors.New("token expired")
			}

			users[i].LastAccess = time.Now()
			c.IndentedJSON(http.StatusOK, users[i])
			log.Println("User updated: ", users[i])
			return &users[i], nil
		}
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "incorrect token"})
	return nil, errors.New("incorrect token")
}

func extendSessionRequest(c *gin.Context) {
	username, err := extendSession(c)

	if err != nil {
		return
	}
	c.IndentedJSON(http.StatusOK, username)
}

// todo: call extendSesssion from joinLobby
func joinLobby(c *gin.Context) {
	token, exists := c.GetPostForm("token")

	if token == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no token supplied"})
		return
	}

	for i, u := range users {
		if u.Token.String() == token {
			if users.checkExpiryAndDelete(i) {
				c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
				return
			}

			users[i].LastAccess = time.Now()
			c.IndentedJSON(http.StatusOK, users[i])
			log.Println("User updated: ", users[i])
			return
		}
	}
}

func main() {
	router := gin.Default()
	router.POST("/newSession/:username", postUsers)
	router.POST("/extendSession/:token", extendSessionRequest)
	router.POST("/joinLobby/:token", joinLobby)

	router.Run("localhost:8080")
}
