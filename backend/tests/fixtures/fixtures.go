package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

// getFixturesDir returns the absolute path to backend/tests/fixtures directory.
func getFixturesDir() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "tests/fixtures"
	}
	return filepath.Dir(currentFile)
}

// LoadUserFixtures loads user fixtures from JSON into the database.
func LoadUserFixtures(db *gorm.DB) ([]models.User, error) {
	filePath := filepath.Join(getFixturesDir(), "users.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users fixture: %w", err)
	}

	var users []models.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users fixture: %w", err)
	}

	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			return nil, fmt.Errorf("failed to insert fixture user %d: %w", users[i].ID, err)
		}
	}

	return users, nil
}

// LoadAssetFixtures loads asset fixtures from JSON into the database.
func LoadAssetFixtures(db *gorm.DB) ([]models.Asset, error) {
	filePath := filepath.Join(getFixturesDir(), "assets.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read assets fixture: %w", err)
	}

	var assets []models.Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, fmt.Errorf("failed to parse assets fixture: %w", err)
	}

	for i := range assets {
		if err := db.Create(&assets[i]).Error; err != nil {
			return nil, fmt.Errorf("failed to insert fixture asset %d: %w", assets[i].ID, err)
		}
	}

	return assets, nil
}
