package billing

type Usage struct{
 FilesThisMonth int
 ReportsThisMonth int
}

func CanUpload(p Plan,u Usage,sizeMB int)bool{
 if sizeMB>p.MaxFileMB{return false}
 if u.FilesThisMonth>=p.MaxFiles{return false}
 return true
}
