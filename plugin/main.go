package main

import (
	"github.com/go-accounting/coa"
	"github.com/go-accounting/deb"
	fdespacecoa "github.com/go-accounting/fde-space-coa"
)

var newSpace func(map[string]interface{}, *string, *string) (interface{}, error)
var newKeyValueStore func(map[string]interface{}, *string) (interface{}, error)
var LoadSymbolFunction func(string, string) (interface{}, error)

func NewStoreAndAccountsRepository(
	storeSettings map[string]interface{},
	accountsRepositorySettings map[string]interface{},
	user *string,
	coaid *string,
) (interface{}, interface{}, error) {
	if newSpace == nil {
		symbol, err := LoadSymbolFunction(storeSettings["PluginFile"].(string), "NewSpace")
		if err != nil {
			return nil, nil, err
		}
		newSpace = symbol.(func(map[string]interface{}, *string, *string) (interface{}, error))
	}
	if newKeyValueStore == nil {
		symbol, err := LoadSymbolFunction(accountsRepositorySettings["PluginFile"].(string), "NewKeyValueStore")
		if err != nil {
			return nil, nil, err
		}
		newKeyValueStore = symbol.(func(map[string]interface{}, *string) (interface{}, error))
	}
	space, err := newSpace(storeSettings, user, coaid)
	if err != nil {
		return nil, nil, err
	}
	keyValueStore, err := newKeyValueStore(accountsRepositorySettings, user)
	if err != nil {
		return nil, nil, err
	}
	return fdespacecoa.NewStoreAndAccountsRepository(
		space.(deb.Space),
		coa.NewCoaRepository(keyValueStore.(coa.KeyValueStore)),
		coaid,
	)
}
