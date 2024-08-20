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

type userlist []user

var users = userlist{}

// implement method to remove item from slice
func (u userlist) remove(s int) userlist {
	return append(u[:s], u[s+1:]...)
}

func (u userlist) checkExpiry(i int) bool {
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

	for i, u := range users {
		if u.Token.String() == token {
			if users.checkExpiry(i) {
				c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
			}

			// add condition to remove user if expired
			users[i].LastAccess = time.Now()
			c.IndentedJSON(http.StatusOK, users[i])
			log.Println("User updated: ", users[i])
			return
		}
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "username taken"})
}

func main() {
	router := gin.Default()
	router.POST("/newSession/:username", postUsers)
	router.POST("/extendSession/:token", extendSession)

	router.Run("localhost:8080")
}
