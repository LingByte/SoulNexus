package storage

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var ErrCacheDbNumsError = errors.New("cache db nums input error, exceed 0-16")
var ErrInvalidDbIndex = errors.New("invalid database index")

type CacheServer struct {
	dbs   []*CacheDb
	dbNum int
}

func NewCacheServer(dbNum int) (*CacheServer, error) {
	if dbNum == 0 {
		dbNum = 16
	}
	if dbNum < 0 || dbNum >= 16 {
		return nil, ErrCacheDbNumsError
	}
	server := &CacheServer{
		dbs:   make([]*CacheDb, dbNum),
		dbNum: dbNum,
	}
	for i := range dbNum {
		server.dbs[i] = NewCacheDb(i)
	}
	return server, nil
}

func (s *CacheServer) GetDb(dbIndex int) (*CacheDb, error) {
	if dbIndex < 0 || dbIndex >= s.dbNum {
		return nil, ErrInvalidDbIndex
	}
	return s.dbs[dbIndex], nil
}
