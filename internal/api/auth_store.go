package api

import "strings"

type authUser struct{ID int64; Email string; Hash string}

func normEmail(v string)string{return strings.ToLower(strings.TrimSpace(v))}
