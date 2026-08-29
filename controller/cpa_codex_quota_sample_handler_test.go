package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestCPACodexQuotaSampleHandlerSchedulingContract(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	optionMapWasNil := common.OptionMap == nil
	if optionMapWasNil {
		common.OptionMap = make(map[string]string)
	}
	originalBaseURL, hadBaseURL := common.OptionMap["console_setting.cpa_base_url"]
	originalKey, hadKey := common.OptionMap["console_setting.cpa_management_key"]
	common.OptionMap["console_setting.cpa_base_url"] = "https://cpa.example.com"
	common.OptionMap["console_setting.cpa_management_key"] = "management-key"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if optionMapWasNil {
			common.OptionMap = nil
			return
		}
		if hadBaseURL {
			common.OptionMap["console_setting.cpa_base_url"] = originalBaseURL
		} else {
			delete(common.OptionMap, "console_setting.cpa_base_url")
		}
		if hadKey {
			common.OptionMap["console_setting.cpa_management_key"] = originalKey
		} else {
			delete(common.OptionMap, "console_setting.cpa_management_key")
		}
	})

	handler := cpaCodexQuotaSampleHandler{}
	assert.Equal(t, model.SystemTaskTypeCPACodexQuota, handler.Type())
	assert.Equal(t, 10*time.Minute, handler.Interval())
	assert.True(t, handler.Enabled())

	common.OptionMapRWMutex.Lock()
	common.OptionMap["console_setting.cpa_management_key"] = ""
	common.OptionMapRWMutex.Unlock()
	assert.False(t, handler.Enabled())
}
