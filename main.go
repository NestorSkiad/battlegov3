package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// to be changed to env var for "prod"
const dbURL = "postgres://postgres:admin@localhost:5432/postgres"
const webServerHost = "localhost"
const webServerPort = "8080"
const webServerAddr = webServerHost + webServerPort
const webServerSecret = "randomenvvar774"

// const sqlTimeFormat   = "2006-01-02 15:04:05-07"
var sqlErrorMessage = gin.H{"message": "Unknown SQL error. Contact Admins. Or don't."}
var ErrSQL = errors.New("SQL Error")
var ErrMissingToken = errors.New("no token supplied")
var ErrExpiredToken = errors.New("token expired")
var ErrInvalidToken = errors.New("token invalid")

// https://github.com/gin-gonic/gin/issues/932#issuecomment-306242400

type env struct {
	db      *pgxpool.Pool
	matches *sync.Map
}

type match struct {
	HostToken, GuestToken uuid.UUID
	GameState             *GameState
}

type user struct {
	Name  string    `json:"name"`
	Token uuid.UUID `json:"token"`
}

func newUser(username string) user {
	return user{
		Name:  username,
		Token: uuid.New()}
}

func (e *env) getUser(token uuid.UUID, c *gin.Context) (*user, error) {
	var username string
	var lastaccess string
	err := e.db.QueryRow(context.Background(), "SELECT username, lastaccess FROM tokens WHERE token = $1", token.String()).Scan(&username, &lastaccess)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return nil, err
	}

	/*
		lastAccessTime, err := time.Parse(sqlTimeFormat, lastaccess)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Time conversion should NOT have failed!!!"})
			return nil, err
		}
	*/

	return &user{Name: username, Token: token}, nil
}

func (e *env) RemoveUser(token uuid.UUID, c *gin.Context) error {
	_, err := e.db.Exec(context.Background(), "DELETE FROM tokens WHERE token = $1", token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return err
	}
	return nil
}

func (e *env) CheckExpiryAndDelete(token uuid.UUID, c *gin.Context) (bool, error) {
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

	_, err = e.db.Exec(context.Background(), "DELETE FROM tokens WHERE token = $1 AND NOW() - lastaccess > INTERVAL '10 minutes'", token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return false, err
	}

	return true, e.RemoveUser(token, c)
}

func (e *env) postUsers(c *gin.Context) {
	username, exists := c.GetPostForm("username")

	if username == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no username supplied"})
		return
	}

	if len(username) > 15 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "username too long"})
		return
	}

	rows, _ := e.db.Query(context.Background(), "SELECT COUNT(*) FROM tokens WHERE username = $1", username)
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
	_, err = tx.Exec(context.Background(), "INSERT INTO tokens (username, token, lastaccess) VALUES ($1, $2, NOW())", newu.Name, newu.Token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	_, err = tx.Exec(context.Background(), "INSERT INTO user_status (user_token, user_status) VALUES ($1, $2)", newu.Token.String(), "idle")
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{"token": newu.Token.String(), "message": "user created"})
}

// TODO: rename to ValidateToken
func (e *env) extendSession(c *gin.Context) (uuid.UUID, error) {
	token, exists := c.GetPostForm("token")

	if token == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no token supplied"})
		return uuid.Nil, ErrMissingToken
	}

	tokenUUID, err := uuid.Parse(token)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "token not UUID"})
		return uuid.Nil, err
	}

	expired, err := e.CheckExpiryAndDelete(tokenUUID, c)
	if err != nil {
		return uuid.Nil, err
	}

	if expired {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "token expired"})
		return uuid.Nil, ErrExpiredToken
	}

	_, err = e.db.Exec(context.Background(), "UPDATE tokens SET lastaccess = NOW() WHERE token = $1", token)
	return tokenUUID, err
}

func (e *env) extendSessionRequest(c *gin.Context) {
	if _, err := e.extendSession(c); err != nil {
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "session extended"})
}

func (e *env) joinMatch(c *gin.Context) {
	guestToken, err := e.extendSession(c)
	if err != nil {
		return
	}

	rows, _ := e.db.Query(context.Background(), "SELECT COUNT(*) FROM user_status WHERE user_status = $1", "hosting")
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if matches == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no match hosts found, consider hosting"})
		return
	}

	var hostTokenString, hostAddrString, hostPortString string
	err = e.db.QueryRow(context.Background(), `
		SELECT
			t.token,
			us.host_addr
		FROM
			user_status AS us,
			tokens AS t
		WHERE us.user_status = $1
			AND NOW() - t.lastaccess < INTERVAL '10 minutes'
			AND us.username = t.username
		ORDER BY RANDOM()
		LIMIT 1
		`, "hosting").Scan(&hostTokenString, &hostAddrString, &hostPortString)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	hostToken, _ := uuid.Parse(hostTokenString)

	_, err = e.db.Exec(context.Background(), "UPDATE user_status SET user_status = $1 WHERE user_token in ($2, $3)", "playing", guestToken, hostToken)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	matchID := uuid.New()
	_, err = e.db.Exec(context.Background(), "INSERT INTO games (uuid, player_one, player_two, host_addr) VALUES ($1, $2, $3, $4)", matchID.String(), hostToken.String(), guestToken.String(), hostAddrString)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if hostAddrString != webServerHost {
		resp, err := http.Get("http://" + hostAddrString)
		if err != nil || resp.StatusCode != http.StatusOK {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal communication error"})
			return
		}
		resp.Body.Close()

		c.IndentedJSON(http.StatusFound, gin.H{"message": "redirect requests to host server", "location": "http://" + hostAddrString})
		return
	}

	gs, err := newGameState()

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "could not create game state"})
	}

	e.matches.Store(matchID, match{HostToken: hostToken, GuestToken: guestToken, GameState: gs})
	c.IndentedJSON(http.StatusOK, gin.H{"message": "successfully joined game", "matchID": matchID.String()})

	// TODO: rest of game logic
	// add list to match
	// one user gets allowed during even turns, the other during odds
	// make a group to handle game requests
	// match functions should run as match dot something dot functions
	// squash the errors first though
}

func (e *env) hostMatch(c *gin.Context) {
	token, err := e.extendSession(c)
	if err != nil {
		return
	}

	tx, err := e.db.Begin(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	defer tx.Rollback(context.Background())

	rows, _ := tx.Query(context.Background(), "SELECT COUNT(*) FROM user_status WHERE user_token = $1 AND user_status = $2", token.String(), "idle")
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if matches == 0 {
		c.IndentedJSON(http.StatusConflict, gin.H{"message": "user not idle"})
		return
	}

	_, err = tx.Exec(context.Background(), "UPDATE user_status SET user_status = $1, host = $2 WHERE user_token = $3", "hosting", webServerHost, token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "user now looking for other players"})
}

func (e *env) unhostMatch(c *gin.Context) {
	token, err := e.extendSession(c)
	if err != nil {
		return
	}

	tx, err := e.db.Begin(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	defer tx.Rollback(context.Background())

	rows, _ := tx.Query(context.Background(), "SELECT COUNT(*) FROM user_status WHERE user_token = $1 AND user_status IN ($2, $3)", token.String(), "idle", "playing")
	matches, err := pgx.CollectOneRow(rows, pgx.RowTo[int32])
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if matches == 1 {
		c.IndentedJSON(http.StatusConflict, gin.H{"message": "user was not hosting"})
		return
	}

	_, err = tx.Exec(context.Background(), "UPDATE user_status SET user_status = $1 WHERE user_token = $2", "idle", token.String())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "user no longer hosting"})
}

// FIXME: naming inconsistencies between games/matches
func (e *env) loadGame(c *gin.Context) {
	gameID, exists := c.GetPostForm("game_id")

	if gameID == "" || !exists { // maybe split
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No game ID supplied. Malformed internal request?!"})
		return
	}

	if _, exists := e.matches.Load(gameID); exists {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "match already in memory"})
		return
	}

	var hostTokenString, guestTokenString string
	err := e.db.QueryRow(context.Background(), `
		SELECT
			player_one,
			player_two
		FROM
			games
		WHERE
			game_id = $1
	`, gameID).Scan(&hostTokenString, guestTokenString)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	hostToken, _ := uuid.Parse(hostTokenString)
	guestToken, _ := uuid.Parse(guestTokenString)

	e.matches.Store(gameID, match{HostToken: hostToken, GuestToken: guestToken})
	c.IndentedJSON(http.StatusOK, gin.H{"message": "match successfully stored in memory"})
}

func (e *env) checkSecret(c *gin.Context) {
	secret, exists := c.GetPostForm("secret")

	if secret == "" || !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No secret supplied. Malformed internal request?!"})
		c.Abort()
		return
	}

	if secret != webServerSecret {
		c.IndentedJSON(http.StatusForbidden, gin.H{"message": "GET OUT"})
		c.Abort()
	}
}

func (e *env) initHost() {
	e.db.Exec(context.Background(), "INSERT INTO hosts (host_addr) VALUES ($1)", webServerAddr)
}

func main() {
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Printf("Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	matches := sync.Map{}
	env := &env{db: dbpool, matches: &matches}
	go env.initHost() // TODO: error handling for this

	router := gin.Default()

	router.POST("/user/:username", env.postUsers)
	router.POST("/extendSession/:token", env.extendSessionRequest)
	router.POST("/joinMatch/:token", env.joinMatch)
	router.POST("/hostMatch/:token", env.hostMatch)
	router.DELETE("/hostmatch/:token", env.unhostMatch)

	internalGroup := router.Group("/internal", env.checkSecret)
	{
		internalGroup.POST("/loadGame", env.loadGame)
	}

	// TODO: user middleware function to send redirect if game on different server (use e.matches, not sql)
	// if redirect needed, call SQL and return host address
	// will need c.Abort() after redirect response
	gameGroup := router.Group("/internal", env.checkSecret)
	{
		gameGroup.GET("/game", env.getMatch)
	}

	router.Run(webServerHost + ":" + webServerPort)
}
