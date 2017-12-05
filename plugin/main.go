package main

import (
	"github.com/go-accounting/coa"
	"github.com/go-accounting/deb"
	fdespacecoa "github.com/go-accounting/fde-space-coa"
)

func NewFdeStoreAndAccountsRepository(config map[string]interface{}, ss ...*string) (interface{}, error) {
	space, err := cast(config["NewSpace"])(config, ss...)
	if err != nil {
		return nil, err
	}
	keyValueStore, err := cast(config["NewKeyValueStore"])(config, ss[0])
	if err != nil {
		return nil, err
	}
	s, ar, err := fdespacecoa.NewStoreAndAccountsRepository(
		space.(deb.Space),
		coa.NewCoaRepository(keyValueStore.(coa.KeyValueStore)),
		ss[1],
	)
	if err != nil {
		return nil, err
	}
	return []interface{}{s, ar}, nil
}

func cast(v interface{}) func(map[string]interface{}, ...*string) (interface{}, error) {
	return v.(func(map[string]interface{}, ...*string) (interface{}, error))
}
