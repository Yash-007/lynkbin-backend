package repo

import (
	"module/lynkbin/internal/models"

	"gorm.io/gorm"
)

type UserRepo struct {
	DB *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{DB: db}
}

func (r *UserRepo) GetUserById(userId int64) (models.User, error) {
	var user models.User
	err := r.DB.Table(user.TableName()).Where("id = ?", userId).First(&user).Error
	return user, err
}

func (r *UserRepo) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.DB.Table(user.TableName()).Where("email=?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) CreateUser(user *models.User) error {
	return r.DB.Table(user.TableName()).Create(user).Error
}

func (r *UserRepo) GetUserIdByEmail(email string) (int64, error) {
	var user models.User
	if err := r.DB.Table(user.TableName()).Where("email=?", email).Select("id").First(&user).Error; err != nil {
		return 0, err
	}
	return *user.Id, nil
}
