package tests

import (
	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/service"

	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) TestCreateUser() {
	err := service.DeleteRecord(&config.Params{DB: t.db}, entity.Entry{GUID: "123"})
	assert.NoError(t.T(), err)
}
