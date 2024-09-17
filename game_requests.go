package main

import "github.com/gin-gonic/gin"

type presentableGameState struct {
	Board *Board `json:"board"`
	Evens PlayerType `json:"firstPlayer"`
	Moves []*Move `json:"moves"`
} 

func (e *env) getGame(_ *gin.Context) {

	// get gameid, token
	// token must be one of tokens in match, which should be in e.matches
	// only return current player's board
	return
}
