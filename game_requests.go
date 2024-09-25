package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type censoredGameState struct {
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

	// put match in var, validate token again but with match tokens
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

	// need toPresentable

	// given token and match id
	//
	return
}
