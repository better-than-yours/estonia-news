package tests

import (
	"testing"

	"estonia-news/db"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type SuiteTest struct {
	suite.Suite
	db *gorm.DB
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTest))
}

func (t *SuiteTest) SetupSuite() {
	t.db = db.Connect("127.0.0.1", "postgres", "postgres", "postgres")
}

func (t *SuiteTest) TearDownSuite() {
	sqlDB, _ := t.db.DB()
	defer sqlDB.Close()
	for _, val := range db.GetModels() {
		t.db.Migrator().DropTable(val)
	}
}

func (t *SuiteTest) SetupTest() {}

func (t *SuiteTest) TearDownTest() {}
