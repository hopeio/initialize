package initialize

import "github.com/hopeio/initialize/conf_center"

type Option func(*globalConfig)

func WithConfigCenter(configCenter ...conf_center.ConfigCenter) Option {
	return func(*globalConfig) {
		for _, cc := range configCenter {
			conf_center.RegisterConfigCenter(cc)
		}
	}
}
