package main

import (
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

var users = []user{}

func newUser(username string) *user {
	user := user{
		Name:       username,
		Token:      uuid.New(),
		LastAccess: time.Now()}

	return &user
}

func getAuthToken(c *gin.Context) {
	username := c.Param("username")

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

func extendSession(c *gin.Context) {
	token := c.Param("token")

	for _, u := range users {
		if u.Token.String() == token {
			u.LastAccess = time.Now()
			c.IndentedJSON(http.StatusOK, u)
			log.Println("Users: ", users)
			return
		}
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "username taken"})
}

func main() {
	router := gin.Default()
	router.POST("/newSession/:username", getAuthToken)
	router.POST("/extendSession/:token", extendSession)

	router.Run("localhost:8080")
}
