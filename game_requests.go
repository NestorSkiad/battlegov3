package main

import "github.com/gin-gonic/gin"

type censoredGameState struct {
	Board *Board `json:"board"`
	Evens PlayerType `json:"firstPlayer"`
	Moves []*Move `json:"moves"`
} 

func (e *env) getGame(_ *gin.Context) {

	// get token
	// token must be one of tokens in match, which should be in e.matches
	// return game ID
	return
}

func (e *env) getGameState(_ *gin.Context) {

	// get game ID from matches

	// get token
	// token must be one of tokens in match, which should be in e.matches
	// return game ID
	return
}
