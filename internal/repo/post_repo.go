package repo

import (
	"errors"
	"module/lynkbin/internal/models"
	"slices"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepo struct {
	DB *gorm.DB
}

func NewPostRepo(db *gorm.DB) *PostRepo {
	return &PostRepo{DB: db}
}

func (r *PostRepo) CreatePost(post *models.Post) error {
	return r.DB.Create(post).Error
}

func (r *PostRepo) CheckUserAuthorExists(userId int64, author string, platform string) (bool, error) {
	if author == "" {
		return false, nil
	}
	err := r.DB.Where("names @> ?", pq.StringArray{author}).Where("user_id = ? AND  platform= ?", userId, platform).First(&models.UserAuthor{}).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostRepo) AddUserAuthor(userId int64, author string, platform string) error {
	if author == "" {
		return nil
	}
	var userAuthor models.UserAuthor
	err := r.DB.Where("user_id = ? AND  platform= ?", userId, platform).First(&userAuthor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.DB.Create(&models.UserAuthor{
				UserId:   userId,
				Platform: platform,
				Names:    pq.StringArray{author},
			}).Error
		}
		return err
	}
	userAuthor.Names = append(userAuthor.Names, author)
	return r.DB.Save(&userAuthor).Error
}

func (r *PostRepo) UpdateUserTags(userId int64, platform string, tags pq.StringArray) error {
	if len(tags) == 0 {
		return nil
	}
	var userTags models.UserTags
	err := r.DB.Where("tags @> ?", tags).Where("user_id = ? AND  platform= ?", userId, platform).First(&userTags).Error

	if err == nil {
		return nil
	}

	err = r.DB.Where("user_id = ? AND  platform= ?", userId, platform).First(&userTags).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.DB.Create(&models.UserTags{
				UserId:   userId,
				Platform: platform,
				Tags:     tags,
			}).Error

			if err != nil {
				return err
			}
			err = r.UpdateAllTags(tags)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	for _, tag := range tags {
		if !slices.Contains(userTags.Tags, tag) {
			userTags.Tags = append(userTags.Tags, tag)
		}
	}
	if len(userTags.Tags) > 0 {
		err = r.DB.Save(&userTags).Error
		if err != nil {
			return err
		}
		err = r.UpdateAllTags(tags)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepo) UpdateAllTags(tags pq.StringArray) error {
	tagModels := make([]models.AllTags, len(tags))

	for i, tag := range tags {
		tagModels[i] = models.AllTags{
			Tag: tag,
		}
	}
	err := r.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&tagModels).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *PostRepo) UpdateUserCategory(userId int64, platform string, category string) error {
	if category == "" {
		return nil
	}
	var userCategories models.UserCategories
	err := r.DB.Where("user_id = ? AND  platform= ?", userId, platform).First(&userCategories).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.DB.Create(&models.UserCategories{
				UserId:     userId,
				Platform:   platform,
				Categories: pq.StringArray{category},
			}).Error

			if err != nil {
				return err
			}
			err = r.UpdateAllCategories(category)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if slices.Contains(userCategories.Categories, category) {
		return nil
	}
	userCategories.Categories = append(userCategories.Categories, category)
	err = r.DB.Save(&userCategories).Error
	if err != nil {
		return err
	}
	err = r.UpdateAllCategories(category)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostRepo) UpdateAllCategories(category string) error {
	err := r.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.AllCategories{
		Category: category,
	}).Error
	return err
}

func (r *PostRepo) GetAllTags() ([]models.AllTags, error) {
	var tags []models.AllTags
	err := r.DB.Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *PostRepo) GetAllCategories() ([]models.AllCategories, error) {
	var categories []models.AllCategories
	err := r.DB.Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *PostRepo) GetPosts(userId int64, platform string, tags []string, authors []string, categories []string) ([]models.Post, error) {
	var posts []models.Post
	query := r.DB.Where("user_id = ?", userId)

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if len(tags) > 0 {
		query = query.Where("tags && ?", pq.StringArray(tags))
	}

	if len(authors) > 0 {
		query = query.Where("author in ?", authors)
	}
	if len(categories) > 0 {
		query = query.Where("category in ?", categories)
	}

	err := query.Order("created_at DESC").Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepo) GetUserAuthors(userId int64, platform string) ([]models.UserAuthor, error) {
	var authors []models.UserAuthor
	err := r.DB.Where("user_id = ? AND platform = ?", userId, platform).Find(&authors).Error
	if err != nil {
		return nil, err
	}
	return authors, nil
}

func (r *PostRepo) GetUserCategories(userId int64, platform string) ([]models.UserCategories, error) {
	var categories []models.UserCategories
	err := r.DB.Where("user_id = ? AND platform = ?", userId, platform).Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *PostRepo) GetUserTags(userId int64, platform string) ([]models.UserTags, error) {
	var tags []models.UserTags
	err := r.DB.Where("user_id = ? AND platform = ?", userId, platform).Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *PostRepo) GetAllUserPostsCount(userId int64) (int64, error) {
	var count int64
	err := r.DB.Model(&models.Post{}).Where("user_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostRepo) GetAllTagsCount(userId int64) (int64, error) {
	var count int64
	err := r.DB.Raw("SELECT SUM(CARDINALITY(tags)) FROM user_tags WHERE user_id = ?", userId).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostRepo) GetAllCategoriesCount(userId int64) (int64, error) {
	var count int64
	err := r.DB.Raw("SELECT SUM(CARDINALITY(categories)) FROM user_categories WHERE user_id = ?", userId).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
