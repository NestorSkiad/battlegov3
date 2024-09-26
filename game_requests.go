package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CensoredGameState contains all information in gameState,
// but only one board. Use gameState.toCensored()
type CensoredGameState struct {
	Board *Board     `json:"board"`
	Evens PlayerType `json:"firstPlayer"`
	Moves []*Move    `json:"moves"`
}

func (e *env) getMatch(c *gin.Context) {
	userToken, err := e.extendSession(c)
	if err != nil {
		return
	}

	var matchID string
	var hostAddr string
	err = e.db.QueryRow(context.Background(), `
		SELECT
			g.game_id
			g.host_addr
		FROM
			games AS g
		WHERE
			EXISTS (
				SELECT
					us.user_token
				FROM
					user_status AS us
				WHERE
					us.token = $1 AND
					us.user_status = 'playing'
			) AND
			($1 = g.player_one OR
			$1 = g.player_two)
	`, userToken.String()).Scan(matchID, hostAddr)

	if errors.Is(err, pgx.ErrNoRows) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "User not in playing state."})
		return
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
		return
	}

	if hostAddr != webServerHost {
		c.IndentedJSON(http.StatusFound, gin.H{"message": "redirect requests to host server", "location": "http://" + hostAddr, "game_id": matchID})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "you are talking to the host server", "game_id": matchID})
}

func (e *env) getGameState(c *gin.Context) {
	userToken, err := e.extendSession(c)
	if err != nil {
		return
	}

	matchTokenString, exists := c.GetPostForm("match_id")
	if !exists || matchTokenString == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no match token supplied"})
		return
	}

	matchTokenUUID, err := uuid.Parse(matchTokenString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "match token not UUID"})
		return
	}

	// TODO: turn this until before censoredGameState into own functions or middleware
	var match *Match
	matchUncast, ok := e.matches.Load(matchTokenUUID)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "could not retrieve match from memory"})
		return
	}

	match = matchUncast.(*Match)

	var p PlayerType
	switch userToken {
	case match.HostToken:
		p = Host
	case match.GuestToken:
		p = Guest
	default:
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "player not in match"})
		return
	}

	censoredGameState := match.GameState.toCensored(p)

	c.IndentedJSON(http.StatusOK, *censoredGameState)
}

func (e *env) postMove(c *gin.Context) {
	userToken, err := e.extendSession(c)
	if err != nil {
		return
	}

	matchTokenString, exists := c.GetPostForm("match_id")
	if !exists || matchTokenString == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no match token supplied"})
		return
	}

	matchTokenUUID, err := uuid.Parse(matchTokenString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "match token not UUID"})
		return
	}

	// TODO: turn this until before censoredGameState into own functions or middleware
	// needs to return player type, match pointer
	var match *Match
	matchUncast, ok := e.matches.Load(matchTokenUUID)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "could not retrieve match from memory"})
		return
	}

	match = matchUncast.(*Match)

	var p PlayerType
	switch userToken {
	case match.HostToken:
		p = Host
	case match.GuestToken:
		p = Guest
	default:
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "player not in match"})
		return
	}

	xString, exists := c.GetPostForm("x")
	if !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "x coord not supplied"})
		return
	}

	yString, exists := c.GetPostForm("y")
	if !exists {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "y coord not supplied"})
		return
	}

	x, err := strconv.Atoi(xString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "x not an integer"})
	}

	y, err := strconv.Atoi(yString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "y not an integer"})
	}

	res, err := match.GameState.tryHitEnemy(x, y, p)

	//check if winner
	// if not return below
	c.IndentedJSON(http.StatusOK, gin.H{"message": "move registered", "hit": res})

	// if yes, return win message, launch thread to delete match from matches and update SQL in 10 minutes
	// todo: check if winner
}
