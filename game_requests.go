package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
	err = e.db.QueryRow(context.Background(),`
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
	`, userToken.String()).Scan(matchID)

	if errors.Is(err, pgx.ErrNoRows) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "User not in playing state."})
		return
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, sqlErrorMessage)
	}

	// if host doesn't match, send match id and redirect
	// if it does, send ok and match id
	return
}

func (e *env) getGameState(_ *gin.Context) {

	// get match ID from matches

	// get token
	// token must be one of tokens in match, which should be in e.matches
	// return match ID
	return
}
