package main

import (
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

func newUser (username string) *user {
	user := user {
		Name: username,
		Token: uuid.New(),
		LastAccess: time.Now()}
	
	return &user
}

func getAuthToken(c *gin.Context) {
	username := c.Param("username")

	for _, u := range users {
		if u.Name == username {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "username taken"})
			return
		}
	}

	newu := newUser(username)
	users = append(users, *newu)
	c.IndentedJSON(http.StatusCreated, newu)
}

func main() {
	router := gin.Default()
	router.GET("/auth/:username", getAuthToken)

	router.Run("localhost:8080")
}
