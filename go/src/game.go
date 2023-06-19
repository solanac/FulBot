package main

type Game struct {
	Id          int
	Active      bool
	Players     []int
	OrganizerID int
	Size        string
	MaxPlayers  int
	Address     []string
	Schedule    []string
	Date        []string
}
