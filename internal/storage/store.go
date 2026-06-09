package storage

import "github.com/jackc/pgx/v5/pgxpool"

type Store struct{db *pgxpool.Pool}

func New(db *pgxpool.Pool)*Store{return &Store{db:db}}
