package db

import "time"

type Task struct {
	ID      int64     `json:"id"`
	Date    time.Time `json:"date"`
	Title   string    `json:"title"`
	Comment string    `json:"comment"`
	Repeat  string    `json:"repeat"`
}
