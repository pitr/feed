package db

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Conn struct {
	db *gorm.DB
}

type User struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	Email     string
	Feeds     []Feed
}

type Feed struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UserID    uint
	User      User
	URL       string
}

func NewConn() *Conn {
	var err error
	db, err := gorm.Open("sqlite3", "feed.db")
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&User{}, &Feed{})

	return &Conn{db: db}
}

func (c *Conn) GetUsers() ([]User, error) {
	var users []User
	return users, c.db.Preload("Feeds").Find(&users).Error
}

func (c *Conn) FindOrCreateUser(email string) (*User, error) {
	var u User

	return &u, c.db.FirstOrCreate(&u, User{Email: email}).Error
}

func (c *Conn) AddFeed(u *User, feed string) error {
	f := &Feed{URL: feed}

	return c.db.Model(u).Association("Feeds").Append(f).Error

}
