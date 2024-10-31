/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package app

import (
	`crypto/md5`
	`encoding/hex`
	"fmt"
	`io`
	`os`

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	ConfigTypeToml = "toml"
	ConfigTypeJson = "json"
	ConfigTypeYaml = "yaml"
)

type Config interface {
	Sub(key string) Config
	Unmarshal(rawVal interface{}) error
	UnmarshalKey(key string, rawVal interface{}) error
}

type WaterConfig interface {
	OnConfigChange(cfg Config)
}

func NewConfig(configPath, configName, configType string, water WaterConfig) (*ConfigImpl, error) {
	var option viper.DecoderConfigOption
	switch configType {
	case ConfigTypeToml:
		option = decoderToml
	case ConfigTypeJson:
		option = decoderJson
	case ConfigTypeYaml:
		option = decoderYaml
	default:
		err := fmt.Errorf("invaild config type is %s", configType)
		return nil, err
	}

	c := &ConfigImpl{"", viper.New(), water, option, ""}
	c.cfg.AddConfigPath(configPath)
	c.cfg.SetConfigName(configName)
	c.cfg.SetConfigType(configType)
	if err := c.ReadConfig(); err != nil {
		return c, err
	}

	return c, nil
}

func NewConfig2(configFile, configType string, water WaterConfig) (*ConfigImpl, error) {
	var option viper.DecoderConfigOption
	switch configType {
	case ConfigTypeToml:
		option = decoderToml
	case ConfigTypeJson:
		option = decoderJson
	case ConfigTypeYaml:
		option = decoderYaml
	default:
		err := fmt.Errorf("invaild config type is %s", configType)
		return nil, err
	}

	c := &ConfigImpl{"", viper.New(), water, option, ""}
	c.cfg.SetConfigFile(configFile)
	if err := c.ReadConfig(); err != nil {
		return c, err
	}

	return c, nil
}

func decoderYaml(c *mapstructure.DecoderConfig) {
	c.TagName = "yaml"
}

func decoderToml(c *mapstructure.DecoderConfig) {
	c.TagName = "toml"
}

func decoderJson(c *mapstructure.DecoderConfig) {
	c.TagName = "json"
}

func GetFileMd5(fileName string) string {
	fd, err := os.Open(fileName)
	if err != nil {
		return ""
	}

	defer func() {
		_ = fd.Close()
	}()

	md5h := md5.New()
	_, _ = io.Copy(md5h, fd)
	return hex.EncodeToString(md5h.Sum(nil))
}

type ConfigImpl struct {
	key    string
	cfg    *viper.Viper
	water  WaterConfig
	option viper.DecoderConfigOption
	md5    string
}

func (c *ConfigImpl) ReadConfig() error {
	err := c.cfg.ReadInConfig()
	if err != nil {
		return err
	}

	c.md5 = GetFileMd5(c.cfg.ConfigFileUsed())
	return nil
}

func (c *ConfigImpl) SetConfig(in string) {
	c.cfg.SetConfigFile(in)
}

func (c *ConfigImpl) Water() {
	c.cfg.OnConfigChange(func(in fsnotify.Event) {
		if c.water != nil {
			md5v := GetFileMd5(c.cfg.ConfigFileUsed())
			if md5v == c.md5 {
				return
			}

			c.md5 = md5v
			c.water.OnConfigChange(c)
		}
	})
	c.cfg.WatchConfig()
}

func (c *ConfigImpl) Sub(key string) Config {
	return &ConfigImpl{key, c.cfg.Sub(key), nil, c.option, ""}
}

func (c *ConfigImpl) Unmarshal(rawVal interface{}) error {
	if c.cfg == nil {
		return fmt.Errorf("while unmarshaling config: %s field is non-existent", c.key)
	}

	return c.cfg.Unmarshal(rawVal, c.option)
}

func (c *ConfigImpl) UnmarshalKey(key string, rawVal interface{}) error {
	if c.cfg == nil {
		return fmt.Errorf("while unmarshaling config: %s field is non-existent", c.key)
	}

	return c.cfg.UnmarshalKey(key, rawVal, c.option)
}
