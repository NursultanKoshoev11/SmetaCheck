package storage

import "context"

func (s *Store) CreateEstimate(ctx context.Context,name string)(string,error){
 var id string
 q:="INSERT INTO estimates(file_name,status) VALUES($1,'queued') RETURNING id"
 err:=s.db.QueryRow(ctx,q,name).Scan(&id)
 return id,err
}
