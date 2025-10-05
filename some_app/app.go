package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func validateName(name string) error {
	if len(name) > 10 {
		return errors.New("invalid name length")
	} // тоже сомнительно но с натяжкой окей
	return nil
}

func validatePhone(phone string) error {
	if len(phone) != 11 {
		return errors.New("invalid phone format")
	} // нуу не все так просто
	// лучше конечно использовать готовые либы с валидатором или накрайняк regexp
	return nil
}

type User struct {
	Id           int       `json:"id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	IsAdmin      bool      `json:"isAdmin"`
	LastViewedAt time.Time `json:"lastViewedAt"`
}

func (u *User) SetLastViewedAt(t time.Time) {
	u.LastViewedAt = t
}

type UserRepository interface {
	Find(sql string) *User
	Save(user *User)
}

type UserApi struct {
	userRepo UserRepository
	logger   *zap.Logger
}

func NewUserApi(userRepo UserRepository) *UserApi {
	api := &UserApi{
		userRepo: userRepo,
	}
	if os.Getenv("LOGGER_PROD") == "1" {
		api.logger, _ = zap.NewProduction()
	} else {
		api.logger, _ = zap.NewDevelopment()
	}
	return api
}

func (api *UserApi) ProcessRequest(c *gin.Context) {
	authUserId := c.GetInt("auth_user_id")
	// может быть 0 как я понял из ф-ии GetInt
	authUser := api.userRepo.Find(fmt.Sprintf("SELECT * FROM user WHERE id = %d", authUserId))
	// репозиторий разве не должен скрывать эту логику внутри?..

	userId := c.Query("id")
	// может быть любой строкой, нужно так же использовать GetInt
	user := api.userRepo.Find(fmt.Sprintf("SELECT * FROM user WHERE id = %s", userId))
	// репозиторий разве не должен скрывать эту логику внутри?.. х2
	// + sql-injection

	// необходимо изменить логику и при получении nil-юзера сразу отдавать 401/404

	// id=0 при использовании ф-ии GetInt считать за nil-ptr
	// хотя с такой имплементацией в джине я бы лучше стд каст использовал

	// при текущей логике при несуществующем пользователе сразу ловим nil deref

	// if user != nil && authUser != nil && (authUser.IsAdmin || authUser.Id == user.Id)
	// либо вынести user != nil; authUser != nil при их получении из репозитория
	if authUser.IsAdmin || authUser.Id == user.Id {
		if user != nil && !user.IsAdmin { // if !authUser.IsAdmin, user - это к кому мы заходим
			user.SetLastViewedAt(time.Now()) // ненужный функционал, time.Now вынести в функцию
		}

		if c.Request.Method == "POST" {
			bytes, _ := io.ReadAll(c.Request.Body) // ex
			var body map[string]string             // десериализация в мапу это не по нашему
			json.Unmarshal(bytes, &body)           // ex

			var errs map[string]string // это тоже конечно сильно
			if err := validateName(body["name"]); err != nil {
				errs["name"] = err.Error()
			} else {
				user.Name = body["name"] // possible nil deref
			}
			if err := validatePhone(body["phone"]); err != nil {
				errs["phone"] = err.Error()
			} else {
				user.Phone = body["phone"] // possible nil deref
			}
			if len(errs) > 0 {
				c.JSON(http.StatusUnprocessableEntity, errs)
				// а return где кек
			}
		}

		// 0 ех логики в репозитории, самой уязвимой части приложения
		api.userRepo.Save(user)
		api.logger.Debug(fmt.Sprintf("user saved %v", user), zap.Int("user_id", user.Id))
		// а зачем нам логгер который только в девелопмент режиме что то пишет ээ ну ок

		c.JSON(http.StatusOK, user)
		return
	}

	c.Status(http.StatusForbidden) // иначе нельзя.. ну, окей
}
