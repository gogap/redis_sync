package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/gogap/errors"
)

type redisConfig struct {
	Address string `json:"address"`
	Db      int    `json:"db"`
	Auth    string `json:"auth"`
}

type valueType struct {
	Key   string `json:"key"`
	Field string `json:"field,omitempty"`
	Type  string `json:"type"`
}

type syncConfig struct {
	Redis      redisConfig `json:"redis"`
	ValueTypes []valueType `json:"value_types"`

	mapTypes map[string]valueType
}

func (p *syncConfig) Load(fileName string) (err error) {

	var data []byte

	if data, err = ioutil.ReadFile(fileName); err != nil {
		err = ERR_LOAD_CONFIG_FAILED.New(errors.Params{"fileName": fileName, "err": err})
		return
	}

	if err = json.Unmarshal(data, p); err != nil {
		err = ERR_PARSE_CONFIG_FAILED.New(errors.Params{"fileName": fileName, "err": err})
		return
	}

	if p.Redis.Address == "" {
		err = ERR_CONFIG_VALUE_MUST_INPUT.New(errors.Params{"configName": "redis.address"})
		return
	}

	if p.ValueTypes != nil &&
		len(p.ValueTypes) > 0 {

		p.mapTypes = make(map[string]valueType)

		for _, vT := range p.ValueTypes {
			if vT.Key == "" {
				err = ERR_REDIS_KEY_IS_EMPTY.New()
				return
			}

			key := vT.Key
			if vT.Field != "" {
				key = key + "-" + vT.Field
			}

			if v, exist := p.mapTypes[key]; exist {
				if v.Field == vT.Field &&
					v.Type == vT.Field &&
					v.Key == vT.Key {
					continue
				}

				err = ERR_KEY_TYPES_MAP_ALREADY_EXIST.New(errors.Params{"key": vT.Key, "field": vT.Field, "type": vT.Type})
				return
			}

			switch vT.Type {
			case "string", "number", "object", "array", "bool":
				{

				}
			default:
				{
					err = ERR_UNSUPPORT_TYPE_MAPPING.New(errors.Params{"key": vT.Key, "field": vT.Field, "type": vT.Type})
					return
				}
			}

			p.mapTypes[key] = vT
		}
	}

	return
}

func (p *syncConfig) Serialize() (str string, err error) {
	var data []byte
	if data, err = json.MarshalIndent(p, "", "    "); err != nil {
		err = ERR_SERIALIZE_CONFIG_FAILED.New(errors.Params{"err": err})
		return
	}

	str = string(data)
	return
}

func (p *syncConfig) KeyType(key string) (string, bool) {
	if strType, exist := p.mapTypes[key]; exist {
		return strType.Type, true
	}

	return "string", false
}

func (p *syncConfig) HKeyType(key, field string) (string, bool) {
	if strType, exist := p.mapTypes[key+"-"+field]; exist {
		return strType.Type, true
	}

	return "string", false
}
