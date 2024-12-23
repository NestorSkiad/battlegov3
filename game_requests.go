package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

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
	userToken := c.MustGet("token").(uuid.UUID)

	var matchID string
	var hostAddr string
	err := e.db.QueryRow(context.Background(), `
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

func (e *env) playAuth(c *gin.Context) {
	userToken := c.MustGet("token").(uuid.UUID)

	matchTokenString, exists := c.GetPostForm("match_id")
	if !exists || matchTokenString == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no match token supplied"})
		c.Abort()
		return
	}

	matchTokenUUID, err := uuid.Parse(matchTokenString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "match token not UUID"})
		c.Abort()
		return
	}

	// TODO: turn this until before censoredGameState into own functions or middleware
	var match *Match
	matchUncast, ok := e.matches.Load(matchTokenUUID)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "could not retrieve match from memory"})
		c.Abort()
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
		c.Abort()
	}

	c.Set("match", match)
	c.Set("playerType", p)
	c.Set("matchToken", matchTokenUUID)
}

func (e *env) getGameState(c *gin.Context) {
	match := c.MustGet("match").(*Match)
	p := c.MustGet("playerType").(PlayerType)
	censoredGameState := match.GameState.toCensored(p)
	c.IndentedJSON(http.StatusOK, *censoredGameState)
}

func (e *env) postMove(c *gin.Context) {
	match := c.MustGet("match").(*Match)
	p := c.MustGet("playerType").(PlayerType)

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
		return
	}

	y, err := strconv.Atoi(yString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "y not an integer"})
		return
	}

	hit, err := match.GameState.tryHitEnemy(x, y, p)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if hit && !match.GameState.anyAliveEnemy(p) {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "match complete, you won!!!", "hit": hit, "win": true})
		go func(){
			time.Sleep(10 * time.Minute)
			e.matchCleanup(c.MustGet("matchToken").(uuid.UUID))
			}()
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "move registered", "hit": hit, "win": false})
}

// reuse in forfeit
// TODO: logging
func (e *env) matchCleanup(matchID uuid.UUID) {
	var match *Match
	matchUncast, loaded := e.matches.LoadAndDelete(matchID)
	if !loaded {
		return
	}
	match = matchUncast.(*Match)
	
	tx, _ := e.db.Begin(context.Background())
	defer tx.Rollback(context.Background())

	var hostwin bool
	if match.Winner == Host {
		hostwin = true
	} else {
		hostwin = false
	}

	tx.Exec(context.Background(), "INSERT INTO game_history (player, game_id, won) values ($1, $2, $3)", match.HostToken.String(), matchID.String(), hostwin)
	tx.Exec(context.Background(), "INSERT INTO game_history (player, game_id, won) values ($1, $2, $3)", match.GuestToken.String(), matchID.String(), !hostwin)
	tx.Exec(context.Background(), "DELETE FROM games WHERE game_id=$1", matchID.String())

	tx.Commit(context.Background())
}
