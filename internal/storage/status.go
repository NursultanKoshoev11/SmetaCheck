package storage

import "context"

func (s *Store) SetStatus(ctx context.Context,id,status string)error{
 _,err:=s.db.Exec(ctx,"UPDATE estimates SET status=$1 WHERE id=$2",status,id)
 return err
}
