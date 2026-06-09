package storage

import "context"

func (s *Store) CreateEstimate(ctx context.Context,name,path string)error{
 _,err:=s.db.Exec(ctx,"insert into estimates(file_name,file_path,status) values($1,$2,$3)",name,path,"queued")
 return err
}
